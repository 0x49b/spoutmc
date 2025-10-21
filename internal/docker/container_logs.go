package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.HideBanner = true

	// Basic middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodOptions},
		AllowHeaders: []string{"Content-Type", "Cache-Control"},
	}))

	e.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/logs/:container", handleLogs)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	e.Logger.Fatal(e.Start(addr))
}

func handleLogs(c echo.Context) error {
	containerID := c.Param("container")
	if containerID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "container path parameter required")
	}

	// Query params
	q := c.QueryParams()
	showStdout := parseBoolDefault(q.Get("stdout"), true)
	showStderr := parseBoolDefault(q.Get("stderr"), true)
	tail := q.Get("tail")   // "", "all", "100", etc.
	since := q.Get("since") // unix seconds or RFC3339
	backfill := tail != "" || since != ""

	// SSE headers
	res := c.Response()
	req := c.Request()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set("Cache-Control", "no-cache")
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering
	res.WriteHeader(http.StatusOK)

	flusher, ok := res.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming not supported by server")
	}

	ctx := req.Context()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "docker client init error: "+err.Error())
	}
	defer cli.Close()

	// Inspect container to decide Attach vs Logs and detect TTY
	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "container not found: "+err.Error())
	}
	isTTY := info.Config != nil && info.Config.Tty
	isRunning := info.ContainerJSONBase != nil &&
		info.ContainerJSONBase.State != nil &&
		info.ContainerJSONBase.State.Running

	// Build reader from either Attach (preferred while running) or Logs (fallback)
	var reader io.Reader // <-- note: io.Reader (not ReadCloser)
	var closer func()

	if isRunning {
		att, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
			Stream: true,
			Stdout: showStdout,
			Stderr: showStderr,
			// If caller asked for backfill, include past logs too.
			// (Attach doesn't support tail/since; Docker will include previous logs if Logs=true)
			Logs: backfill,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "attach failed: "+err.Error())
		}
		reader = att.Reader // *bufio.Reader (no Close)
		//closer = func() { _ = att.Close() }    // Close the hijacked connection when done
	} else {
		// Stopped: use Logs; will end after sending (Follow has no effect if container is stopped)
		opts := container.LogsOptions{
			ShowStdout: showStdout,
			ShowStderr: showStderr,
			Follow:     true,
			Timestamps: false,
			Details:    false,
		}
		if tail != "" {
			opts.Tail = tail
		}
		if since != "" {
			if _, err := strconv.ParseInt(since, 10, 64); err != nil {
				if t, perr := time.Parse(time.RFC3339, since); perr == nil {
					opts.Since = strconv.FormatInt(t.Unix(), 10)
				} else {
					return echo.NewHTTPError(http.StatusBadRequest, "invalid since; use unix seconds or RFC3339")
				}
			} else {
				opts.Since = since
			}
		}
		logReader, err := cli.ContainerLogs(ctx, containerID, opts)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "logs failed: "+err.Error())
		}
		reader = logReader // io.ReadCloser implements io.Reader
		closer = func() { _ = logReader.Close() }
	}
	defer func() {
		if closer != nil {
			closer()
		}
	}()

	// Demultiplex: if TTY=true, stream is raw (stdout only). If TTY=false, use stdcopy to split.
	var stdoutR, stderrR io.Reader
	if isTTY {
		if showStdout {
			stdoutR = reader
		}
		// no separate stderr stream in TTY mode
	} else {
		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		go func() {
			defer outW.Close()
			defer errW.Close()
			_, _ = stdcopy.StdCopy(outW, errW, reader)
		}()
		if showStdout {
			stdoutR = outR
		} else {
			// if not consuming, still drain to avoid blocking
			go io.Copy(io.Discard, outR)
		}
		if showStderr {
			stderrR = errR
		} else {
			go io.Copy(io.Discard, errR)
		}
	}

	// Heartbeats keep proxies from closing idle SSE connections
	heartbeatStop := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-heartbeatStop:
				return
			case <-t.C:
				fmt.Fprint(res, ": ping\n\n")
				flusher.Flush()
			}
		}
	}()

	// Start scanners
	outDone := make(chan struct{})
	errDone := make(chan struct{})
	if stdoutR != nil {
		go streamScanner(ctx, "stdout", stdoutR, res, flusher, outDone)
	} else {
		close(outDone)
	}
	if stderrR != nil {
		go streamScanner(ctx, "stderr", stderrR, res, flusher, errDone)
	} else {
		close(errDone)
	}

	// Wait for client cancellation or both streams to finish
	select {
	case <-ctx.Done():
	case <-outDone:
		<-errDone
	case <-errDone:
		<-outDone
	}

	close(heartbeatStop)
	flusher.Flush()
	return nil
}

func streamScanner(ctx context.Context, event string, r io.Reader, res *echo.Response, flusher http.Flusher, done chan<- struct{}) {
	defer close(done)
	sc := bufio.NewScanner(r)
	// Increase buffer to handle longer log lines (up to 1MB)
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if !sc.Scan() {
			return
		}
		line := sc.Text()
		line = strings.ReplaceAll(line, "\r", "\\r")

		payload, _ := json.Marshal(map[string]any{
			"ts":   time.Now().UTC().Format(time.RFC3339Nano),
			"line": line,
		})
		writeSSE(res, event, string(payload))
		flusher.Flush()
	}
}

func writeSSE(res *echo.Response, event, data string) {
	if event != "" {
		fmt.Fprintf(res, "event: %s\n", event)
	}
	for _, l := range strings.Split(data, "\n") {
		fmt.Fprintf(res, "data: %s\n", l)
	}
	fmt.Fprint(res, "\n")
}

func parseBoolDefault(s string, def bool) bool {
	if s == "" {
		return def
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return b
}
