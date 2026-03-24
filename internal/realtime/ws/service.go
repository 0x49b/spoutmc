package ws

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"spoutmc/internal/docker"
	"spoutmc/internal/log"

	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

var logger = log.GetLogger(log.ModuleAPI)

type ClientMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel,omitempty"`
	Command string `json:"command,omitempty"`
}

type ServerMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type Service struct {
	activeConnections atomic.Int64
}

func NewService() *Service {
	return &Service{}
}

type serverSocket struct {
	conn           *websocket.Conn
	containerID    string
	writeCh        chan ServerMessage
	subscribeStats atomic.Bool
	subscribeLogs  atomic.Bool
	cancelLogs     context.CancelFunc
	logsRunning    atomic.Bool
	statsRunning   atomic.Bool
}

func (s *Service) HandleConnection(ctx context.Context, conn *websocket.Conn, containerID string, userID uint) error {
	socket := &serverSocket{
		conn:        conn,
		containerID: containerID,
		writeCh:     make(chan ServerMessage, 64),
	}

	active := s.activeConnections.Add(1)
	logger.Info("ws_connected",
		zap.String("container", TrimContainerID(containerID)),
		zap.Uint("user_id", userID),
		zap.Int64("active_connections", active))

	closed := make(chan struct{})
	go socket.writeLoop(closed)

	socket.enqueue(ServerMessage{
		Type:      "connected",
		Timestamp: time.Now().Unix(),
	})

	err := socket.readLoop(ctx)

	socket.stopLogs()
	active = s.activeConnections.Add(-1)
	close(socket.writeCh)
	<-closed
	_ = conn.Close()

	logger.Info("ws_disconnected",
		zap.String("container", TrimContainerID(containerID)),
		zap.Uint("user_id", userID),
		zap.String("reason", DisconnectReason(err)),
		zap.Int64("active_connections", active))

	return err
}

func (s *serverSocket) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var msg ClientMessage
		if err := websocket.JSON.Receive(s.conn, &msg); err != nil {
			return err
		}

		switch msg.Type {
		case "subscribe":
			s.handleSubscribe(msg.Channel)
		case "unsubscribe":
			s.handleUnsubscribe(msg.Channel)
		case "command":
			s.handleCommand(ctx, msg.Command)
		case "ping":
			s.enqueue(ServerMessage{Type: "pong", Timestamp: time.Now().Unix()})
		default:
			s.enqueue(ServerMessage{
				Type:      "error",
				Timestamp: time.Now().Unix(),
				Error:     "Unsupported message type",
			})
		}
	}
}

func (s *serverSocket) writeLoop(closed chan<- struct{}) {
	defer close(closed)
	for msg := range s.writeCh {
		if err := websocket.JSON.Send(s.conn, msg); err != nil {
			return
		}
	}
}

func (s *serverSocket) enqueue(msg ServerMessage) {
	select {
	case s.writeCh <- msg:
	default:
		logger.Warn("ws_backpressure_drop",
			zap.String("container", TrimContainerID(s.containerID)),
			zap.String("type", msg.Type))
	}
}

func (s *serverSocket) handleSubscribe(channel string) {
	switch channel {
	case "stats":
		if s.subscribeStats.Load() {
			return
		}
		s.subscribeStats.Store(true)
		logger.Info("ws_subscription_started",
			zap.String("container", TrimContainerID(s.containerID)),
			zap.String("channel", channel))
		go s.runStatsStream()
	case "logs":
		if s.subscribeLogs.Load() {
			return
		}
		s.subscribeLogs.Store(true)
		logger.Info("ws_subscription_started",
			zap.String("container", TrimContainerID(s.containerID)),
			zap.String("channel", channel))
		s.startLogsStream()
	default:
		s.enqueue(ServerMessage{
			Type:      "error",
			Timestamp: time.Now().Unix(),
			Error:     "Unsupported subscribe channel",
		})
	}
}

func (s *serverSocket) handleUnsubscribe(channel string) {
	switch channel {
	case "stats":
		s.subscribeStats.Store(false)
		logger.Info("ws_subscription_stopped",
			zap.String("container", TrimContainerID(s.containerID)),
			zap.String("channel", channel))
	case "logs":
		s.subscribeLogs.Store(false)
		s.stopLogs()
		logger.Info("ws_subscription_stopped",
			zap.String("container", TrimContainerID(s.containerID)),
			zap.String("channel", channel))
	}
}

func (s *serverSocket) runStatsStream() {
	if !s.statsRunning.CompareAndSwap(false, true) {
		return
	}
	defer s.statsRunning.Store(false)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for s.subscribeStats.Load() {
		stats, err := docker.GetContainerStats(context.Background(), s.containerID)
		if err != nil {
			errText := strings.ToLower(err.Error())
			if !errors.Is(err, context.Canceled) && !strings.Contains(errText, "context canceled") {
				s.enqueue(ServerMessage{
					Type:      "error",
					Channel:   "stats",
					Timestamp: time.Now().Unix(),
					Error:     err.Error(),
				})
			}
		} else {
			s.enqueue(ServerMessage{
				Type:      "stats",
				Channel:   "stats",
				Timestamp: time.Now().Unix(),
				Payload:   stats,
			})
		}

		<-ticker.C
	}
}

func (s *serverSocket) startLogsStream() {
	if !s.logsRunning.CompareAndSwap(false, true) {
		return
	}

	logsCtx, cancel := context.WithCancel(context.Background())
	s.cancelLogs = cancel

	go func() {
		defer s.logsRunning.Store(false)
		defer cancel()

		logChan, err := docker.FetchDockerLogs(logsCtx, s.containerID)
		if err != nil {
			s.enqueue(ServerMessage{
				Type:      "error",
				Channel:   "logs",
				Timestamp: time.Now().Unix(),
				Error:     err.Error(),
			})
			return
		}

		for s.subscribeLogs.Load() {
			logline, ok := <-logChan
			if !ok {
				return
			}
			s.enqueue(ServerMessage{
				Type:      "log",
				Channel:   "logs",
				Timestamp: time.Now().Unix(),
				Payload:   logline,
			})
		}
	}()
}

func (s *serverSocket) stopLogs() {
	if s.cancelLogs != nil {
		s.cancelLogs()
		s.cancelLogs = nil
	}
}

func (s *serverSocket) handleCommand(ctx context.Context, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		s.enqueue(ServerMessage{
			Type:      "command_ack",
			Timestamp: time.Now().Unix(),
			Error:     "Command is required",
		})
		return
	}

	if err := docker.ExecuteCommand(ctx, s.containerID, command); err != nil {
		s.enqueue(ServerMessage{
			Type:      "command_ack",
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
		})
		return
	}

	s.enqueue(ServerMessage{
		Type:      "command_ack",
		Timestamp: time.Now().Unix(),
		Payload: map[string]string{
			"status":  "success",
			"message": "Command executed successfully",
			"command": command,
		},
	})
}

func TrimContainerID(containerID string) string {
	if len(containerID) <= 12 {
		return containerID
	}
	return containerID[:12]
}

func DisconnectReason(err error) string {
	if err == nil {
		return "client_closed"
	}
	errText := strings.ToLower(err.Error())
	if errors.Is(err, io.EOF) || strings.Contains(errText, "eof") {
		return "client_closed_eof"
	}
	if errors.Is(err, context.Canceled) || strings.Contains(errText, "context canceled") {
		return "context_canceled"
	}
	return err.Error()
}
