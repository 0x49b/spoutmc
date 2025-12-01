package docker

import (
	"bufio"
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
)

// ExecCommand listens for commands and sends their output/results back via a channel.
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

				// Remove leading slash if present
				cmdToExecute := cmd
				if len(cmdToExecute) > 0 && cmdToExecute[0] == '/' {
					cmdToExecute = cmdToExecute[1:]
				}

				execCreateResp, err := cli.ContainerExecCreate(ctx, containerId, container.ExecOptions{
					Cmd:          []string{"rcon-cli", cmdToExecute},
					AttachStdout: true,
					AttachStderr: true,
				})
				if err != nil {
					outputChan <- fmt.Sprintf("exec create error: %v", err)
					return
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

// ExecuteCommand executes a single command in a container using rcon-cli
// Commands can start with or without a leading slash (/)
func ExecuteCommand(ctx context.Context, containerId string, command string) error {
	// Remove leading slash if present (Minecraft commands can optionally start with /)
	// rcon-cli doesn't require the slash
	if len(command) > 0 && command[0] == '/' {
		command = command[1:]
	}

	// Use rcon-cli which is enabled by default in itzg/minecraft-server
	execCreateResp, err := cli.ContainerExecCreate(ctx, containerId, container.ExecOptions{
		Cmd:          []string{"rcon-cli", command},
		AttachStdout: false,
		AttachStderr: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	// Start the exec without waiting for output
	err = cli.ContainerExecStart(ctx, execCreateResp.ID, container.ExecStartOptions{
		Detach: true,
	})
	if err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	return nil
}
