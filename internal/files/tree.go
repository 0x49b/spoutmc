package files

import (
	"fmt"
	"os"
	"path/filepath"
	"spoutmc/internal/config"
	"spoutmc/internal/log"

	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleFiles)

type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Size     int64       `json:"size,omitempty"`
	ModTime  string      `json:"modTime,omitempty"`
	Children []*FileNode `json:"children,omitempty"`
}

func BuildFileTree(basePath, currentPath string, isRoot bool) (*FileNode, error) {
	info, err := os.Stat(currentPath)
	if err != nil {
		return nil, err
	}

	if !isRoot && ShouldExclude(info.Name()) {
		logger.Debug("Excluding file/folder",
			zap.String("name", info.Name()),
			zap.String("path", currentPath))
		return nil, fmt.Errorf("excluded by pattern")
	}

	relPath, err := filepath.Rel(basePath, currentPath)
	if err != nil {
		relPath = currentPath
	}
	if relPath == "." {
		relPath = ""
	}

	node := &FileNode{
		Name:    info.Name(),
		Path:    relPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}

	if info.IsDir() {
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return node, nil // Return directory node even if we can't read it
		}

		node.Children = make([]*FileNode, 0)
		for _, entry := range entries {
			childPath := filepath.Join(currentPath, entry.Name())
			childNode, err := BuildFileTree(basePath, childPath, false)
			if err != nil {
				logger.Debug("Skipping child",
					zap.String("name", entry.Name()),
					zap.Error(err))
				continue // Skip files we can't read or are excluded
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

func ShouldExclude(name string) bool {
	cfg := config.All()

	if cfg.Files == nil {
		logger.Debug("Files config is nil")
		return false
	}

	if len(cfg.Files.ExcludePatterns) == 0 {
		logger.Debug("No exclusion patterns configured")
		return false
	}

	logger.Debug("Checking exclusion patterns",
		zap.Int("pattern_count", len(cfg.Files.ExcludePatterns)),
		zap.String("checking_name", name))

	for _, pattern := range cfg.Files.ExcludePatterns {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			logger.Debug("Pattern match error",
				zap.String("pattern", pattern),
				zap.String("name", name),
				zap.Error(err))
			if pattern == name {
				logger.Debug("Exact match found",
					zap.String("pattern", pattern),
					zap.String("name", name))
				return true
			}
			continue
		}
		if matched {
			logger.Debug("Pattern matched",
				zap.String("pattern", pattern),
				zap.String("name", name))
			return true
		}
	}

	return false
}
