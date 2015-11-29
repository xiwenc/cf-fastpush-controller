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
	"io/ioutil"
	"regexp"
	"time"

	"github.com/matryer/runner"
	"github.com/xiwenc/cf-fastpush-controller/utils"
)

type FileEntry struct {
	Checksum string
	Modification int64
	Content	 []byte
}

type Status struct {
	Health	 string
}

var task *runner.Task
var cmd *exec.Cmd
var lock = sync.RWMutex{}
var cmdRaw = ""
var store = map[string]*FileEntry{}

const (
	ENV_RESTART_REGEX = "FASTPUSH_RESTART_REGEX"
	ENV_IGNORE_REGEX = "FASTPUSH_IGNORE_REGEX"
	ENV_APP_DIRS = "FASTPUSH_APP_DIRS"
)

func RestartApp(backendRunCommand string) Status {
	log.Println("Restarting backend")

	if len(backendRunCommand) > 0 {
		cmdRaw = backendRunCommand
	} else {
		backendRunCommand = cmdRaw
	}

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
			time.Sleep(1000 * time.Millisecond)
		}
		return nil
	})
	lock.RUnlock()
	return Status{Health: "Restarting"}
}

func ListFiles() map[string]*FileEntry {
	for _, dir := range GetAppDirs() {
		err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
			if f.IsDir() {
				return nil
			}
			if store[path] != nil && store[path].Modification == f.ModTime().Unix() {
				// cache hit
				return nil
			}
			fileEntry := FileEntry{}
			checksum, _ := utils.ChecksumsForFile(path)
			fileEntry.Checksum = checksum.SHA256
			fileEntry.Modification = f.ModTime().Unix()
			lock.RLock()
			store[path] = &fileEntry
			lock.RUnlock()
			return nil
		})
		if err != nil {
			log.Println(err)
		}
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

func UploadFiles(files map[string]*FileEntry) Status {
	status := Status{}
	failed := 0
	updated := 0
	restart := false
	for path, fileEntry := range files {
		log.Println("Updating file: " + path)
		err := ioutil.WriteFile(path, fileEntry.Content, 0644)
		if err != nil {
			log.Println(err)
			failed++
		} else {
			if NeedsRestart(path) {
				restart = true
			}
			updated++
		}
	}

	if failed > 0 {
		status.Health = "Failed to update " + strconv.Itoa(failed) + " files"
	} else {
		status.Health = "Updated " + strconv.Itoa(updated) + " files"
	}

	if restart {
		RestartApp("")
		status.Health = "Restarting after updating " + strconv.Itoa(updated) + " files"
	}
	return status
}


func NeedsRestart(path string) bool {
	ignoreRegex := os.Getenv(ENV_IGNORE_REGEX)
	if len(ignoreRegex) > 0 {
		match, _ := regexp.MatchString(ignoreRegex, path)
		if match {
			log.Println("Skipping restart for: " + path)
			return false
		}
	}
	restartRegex := os.Getenv(ENV_RESTART_REGEX)
	if len(restartRegex) > 0 {
		match, _ := regexp.MatchString(restartRegex, path)
		if match {
			log.Println("Requires restart for: " + path)
			return true
		}
	}
	return false
}

func GetAppDirs() []string {
	appDirsRaw := os.Getenv(ENV_APP_DIRS)
	if len(appDirsRaw) > 0 {
		return strings.Fields(appDirsRaw)
	}
	return []string{"./"}
}