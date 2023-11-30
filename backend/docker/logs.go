package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func ReadLogs(containerName string) error {

	logger.Info(fmt.Sprintf("Try reading logs for %s", containerName))

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	c, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return err
	}

	/**
	container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      opts.since,
		Until:      opts.until,
		Timestamps: opts.timestamps,
		Follow:     opts.follow,
		Tail:       opts.tail,
		Details:    opts.details,
	}
	*/

	responseBody, err := cli.ContainerLogs(ctx, c.ID, types.ContainerLogsOptions{Follow: true})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	/*	if c.Config.Tty {
			_, err = io.Copy( cli, responseBody)
		} else {
			_, err = stdcopy.StdCopy(dockerCli.Out(), dockerCli.Err(), responseBody)
		}*/

	return err
}
