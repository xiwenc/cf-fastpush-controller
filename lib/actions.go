package lib

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"sync"
	"strconv"

	"github.com/matryer/runner"
	"github.com/xiwenc/go-berlin/utils"
	"io/ioutil"
)

type FileEntry struct {
	Path     string
	Checksum string
	Content	 []byte
}

type Status struct {
	Health	 string
}

var task *runner.Task
var cmd *exec.Cmd
var lock = sync.RWMutex{}

func RestartApp(backendRunCommand string) Status {
	log.Println("Restarting backend")
	parts := strings.Fields(backendRunCommand)
	head := parts[0]
	parts = parts[1:len(parts)]
	lock.RLock()

	if task != nil {
		task.Stop()
		cmd.Wait()
	}
	cmd = exec.Command(head, parts...)

	task = runner.Go(func(shouldStop runner.S) error {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()

		for {
			if shouldStop() {
				cmd.Process.Signal(syscall.SIGTERM)
				break
			}
		}
		return nil
	})
	lock.RUnlock()
	return Status{Health: "Restarting"}
}

func ListFiles() []*FileEntry {
	var store = []*FileEntry{}
	dir := "./"

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		fileEntry := FileEntry{}
		fileEntry.Path = path
		checksum, _ := utils.ChecksumsForFile(path)
		fileEntry.Checksum = checksum.SHA256
		store = append(store, &fileEntry)
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return store
}

func GetStatus() Status {
	status := Status{}

	if cmd.Process.Pid > 0 {
		status.Health = "Running"
	} else {
		status.Health = "Not-Running"
	}
	return status
}

func UploadFiles(files []FileEntry) Status {
	status := Status{}
	failed := 0
	updated := 0
	for _, fileEntry := range files {
		log.Println("Updating file: " + fileEntry.Path)
		err := ioutil.WriteFile(fileEntry.Path, fileEntry.Content, 0644)
		if err != nil {
			log.Println(err)
			failed++
		} else {
			updated++
		}
	}

	if failed > 0 {
		status.Health = "Failed to update " + strconv.Itoa(failed) + " files"
	} else {
		status.Health = "Updated " + strconv.Itoa(updated) + " files"
	}
	return status
}