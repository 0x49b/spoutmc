package plugins

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func CheckForServerTap(mountPath string) (bool, error) {
	if _, err := os.Stat(filepath.Join(mountPath, "plugins", "ServerTap.jar")); err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func DownloadServerTap(mountPath string) error {
	// Create the file
	// todo move filename and plugin dir to a config file
	out, err := os.Create(filepath.Join(mountPath, "plugins", "ServerTap.jar"))
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	// Todo move the url to a config file
	resp, err := http.Get("https://github.com/servertap-io/servertap/releases/download/v0.6.1/ServerTap-0.6.1.jar")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
