package watchdog

import (
	"os"
	"path/filepath"
	"spoutmc/internal/log"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

func StartFileWatcher() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("📁 error creating file watcher", zap.Error(err))
		return
	}
	defer func(watcher *fsnotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			logger.Error("📁 error closing file watcher", zap.Error(err))
		}
	}(watcher)

	logger.Info("📁 fileWatcher started")

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logger.Info("📁 file event detected", zap.String("event", event.String()))
				if event.Has(fsnotify.Write) {
					logger.Info("modified file:", zap.String("file", event.Name))
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("📁 file watcher error", zap.Error(err))

			}
		}

	}()

	wd, err := os.Getwd()
	if err != nil {
		logger.Error("📁 error getting working directory", zap.Error(err))
	}
	candidates := []string{
		filepath.Join(wd, "config", "spoutmc.yaml"),
		filepath.Join(wd, "config", "spoutmc.yml"),
	}
	var usedPath string
	for _, candidate := range candidates {
		if _, statErr := os.Stat(candidate); statErr == nil {
			usedPath = candidate
			break
		}
	}

	if usedPath == "" {
		logger.Error("no config file found (looked for spout-servers.yaml/.yml)")
	}

	err = watcher.Add(usedPath)
	if err != nil {
		logger.Error("📁 error adding file to watcher", zap.Error(err))
	}
	<-make(chan struct{})

}
