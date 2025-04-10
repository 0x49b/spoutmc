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

				execCreateResp, err := cli.ContainerExecCreate(ctx, containerId, container.ExecOptions{
					Cmd:          []string{"mc-send-to-console", cmd},
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
