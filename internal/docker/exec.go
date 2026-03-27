package docker

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

func ExecCommand(ctx context.Context, containerId string, cmdChan <-chan string) <-chan string {
	outputChan := make(chan string)

	go func() {
		defer close(outputChan)

		for {
			select {
			case <-ctx.Done():
				outputChan <- fmt.Sprintf("Command timeout: %v", ctx.Err())
				return
			case cmd, ok := <-cmdChan:
				if !ok {
					return
				}

				execCreateResp, err := cli.ContainerExecCreate(ctx, containerId, container.ExecOptions{
					Cmd:          buildConsoleCommand(cmd),
					User:         "1000:1000",
					AttachStdout: true,
					AttachStderr: true,
				})
				if err != nil {
					execCreateResp, err = cli.ContainerExecCreate(ctx, containerId, container.ExecOptions{
						Cmd:          buildRCONCommand(cmd),
						AttachStdout: true,
						AttachStderr: true,
					})
					if err != nil {
						outputChan <- fmt.Sprintf("exec create error (console and rcon fallback): %v", err)
						return
					}
				}

				resp, err := cli.ContainerExecAttach(ctx, execCreateResp.ID, container.ExecStartOptions{
					Detach: false,
				})
				if err != nil {
					outputChan <- fmt.Sprintf("exec attach error: %v", err)
					return
				}
				defer resp.Close()

				scanner := bufio.NewScanner(resp.Reader)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						outputChan <- fmt.Sprintf("Command timeout while reading output: %v", ctx.Err())
						return
					default:
						outputChan <- scanner.Text()
					}
				}
			}
		}
	}()

	return outputChan
}

func ExecuteCommand(ctx context.Context, containerId string, command string) error {
	err := runExecAndWait(ctx, containerId, buildConsoleCommand(command), "1000:1000")
	if err != nil {
		fallbackErr := runExecAndWait(ctx, containerId, buildRCONCommand(command), "")
		if fallbackErr != nil {
			return fmt.Errorf("command execution failed (console path: %v, rcon fallback: %w)", err, fallbackErr)
		}
		return nil
	}

	return nil
}

func runExecAndWait(ctx context.Context, containerID string, cmd []string, user string) error {
	execCreateResp, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		User:         user,
		AttachStdout: false,
		AttachStderr: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	if err := cli.ContainerExecStart(ctx, execCreateResp.ID, container.ExecStartOptions{Detach: true}); err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("command context canceled: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
			inspect, err := cli.ContainerExecInspect(ctx, execCreateResp.ID)
			if err != nil {
				return fmt.Errorf("failed to inspect exec: %w", err)
			}
			if !inspect.Running {
				if inspect.ExitCode != 0 {
					return fmt.Errorf("command exited with code %d", inspect.ExitCode)
				}
				return nil
			}
		}
	}
}

func buildConsoleCommand(command string) []string {
	cmd := cleanMinecraftCommand(command)
	if cmd == "" {
		return []string{"mc-send-to-console"}
	}

	return []string{"mc-send-to-console", cmd}
}

func buildRCONCommand(command string) []string {
	cmd := cleanMinecraftCommand(command)
	if cmd == "" {
		return []string{"rcon-cli"}
	}
	return []string{"rcon-cli", cmd}
}

func cleanMinecraftCommand(command string) string {
	cmd := strings.TrimSpace(command)
	if strings.HasPrefix(cmd, "/") {
		cmd = strings.TrimPrefix(cmd, "/")
	}
	return strings.TrimSpace(cmd)
}
