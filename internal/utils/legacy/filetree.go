package legacyutils

import (
	"os"
	"path/filepath"
	"spoutmc/internal/log"
	"time"
)

type File struct {
	ModifiedTime time.Time `json:"modifiedtime"`
	IsLink       bool      `json:"islink"`
	IsDir        bool      `json:"isdir"`
	LinksTo      string    `json:"linksto"`
	Size         int64     `json:"size"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Children     []*File   `json:"children"`
}

func ToFile(file os.DirEntry, path string) *File {
	fileInfo, err := file.Info()
	if err != nil {
		log.HandleError(err)
		return &File{
			IsDir:    file.IsDir(),
			Name:     file.Name(),
			Path:     path,
			Children: []*File{},
		}
	}

	jsonFile := File{
		ModifiedTime: fileInfo.ModTime(),
		IsDir:        file.IsDir(),
		Size:         fileInfo.Size(),
		Name:         file.Name(),
		Path:         path,
		Children:     []*File{},
	}
	if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		jsonFile.IsLink = true
		jsonFile.LinksTo, _ = filepath.EvalSymlinks(filepath.Join(path, file.Name()))
	}
	return &jsonFile
}

func FileToJSON(path string) *File {
	rootOSFile, _ := os.ReadDir(path)
	rootFile := ToFile(rootOSFile[0], path)
	stack := []*File{rootFile}

	for len(stack) > 0 {
		file := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		children, _ := os.ReadDir(file.Path)
		for _, childEntry := range children {
			child := ToFile(childEntry, filepath.Join(file.Path, childEntry.Name()))
			file.Children = append(file.Children, child)
			stack = append(stack, child)
		}
	}
	return rootFile
}
