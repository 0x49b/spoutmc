package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
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

func ToFile(file os.FileInfo, path string) *File {
	JSONFile := File{ModifiedTime: file.ModTime(),
		IsDir:    file.IsDir(),
		Size:     file.Size(),
		Name:     file.Name(),
		Path:     path,
		Children: []*File{},
	}
	if file.Mode()&os.ModeSymlink == os.ModeSymlink {
		JSONFile.IsLink = true
		JSONFile.LinksTo, _ = filepath.EvalSymlinks(filepath.Join(path, file.Name()))
	} // Else case is the zero values of the fields
	return &JSONFile
}
func FileToJSON(path string) *File {
	rootOSFile, _ := os.Stat(path)
	rootFile := ToFile(rootOSFile, path) //start with root file
	stack := []*File{rootFile}

	for len(stack) > 0 { //until stack is empty,
		file := stack[len(stack)-1] //pop entry from stack
		stack = stack[:len(stack)-1]
		children, _ := ioutil.ReadDir(file.Path) //get the children of entry
		for _, chld := range children {          //for each child
			child := ToFile(chld, filepath.Join(file.Path, chld.Name())) //turn it into a File object
			file.Children = append(file.Children, child)                 //append it to the children of the current file popped
			stack = append(stack, child)                                 //append the child to the stack, so the same process can be run again
		}
	}
	return rootFile
}
