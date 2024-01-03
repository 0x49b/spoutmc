package docker

import (
	"github.com/docker/docker/api/types"
)

func ExecCommand(containerId string, minecraftCommand string) (int, error) {

	// Prepare the exec create options
	createResp, err := cli.ContainerExecCreate(ctx, containerId, types.ExecConfig{
		Cmd: []string{"mc-send-to-console", minecraftCommand},
	})
	if err != nil {
		return -1, err
	}

	// Execute the command in the container
	err = cli.ContainerExecStart(ctx, createResp.ID, types.ExecStartCheck{})
	if err != nil {
		return -1, err
	}

	// Wait for the command to complete (optional)
	waitResp, err := cli.ContainerExecInspect(ctx, createResp.ID)
	if err != nil {
		return -1, err
	}
	return waitResp.ExitCode, nil
}
