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

	"github.com/spf13/viper"
	"github.com/matryer/runner"
	"github.com/xiwenc/cf-fastpush-controller/utils"
	"fmt"
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

func RestartApp(backendRunCommand string) Status {
	log.Println("Restarting backend")

	if len(backendRunCommand) > 0 {
		cmdRaw = backendRunCommand
	} else {
		backendRunCommand = cmdRaw
	}
	log.Println("Backend command: " + backendRunCommand)

	parts := strings.Fields(backendRunCommand)
	head := parts[0]
	parts = parts[1:len(parts)]
	lock.RLock()

	if task != nil {
		log.Println("Stopping running backend")
		task.Stop()
		cmd.Wait()
	}
	cmd = exec.Command(head, parts...)
	// Copy and change current environment for the backend
	cmd.Env = GetBackendEnvironment()

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
		log.Println("Listing files for: " + dir)
		err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
			if f.IsDir() {
				return nil
			}
			if store[path] != nil && store[path].Modification == f.ModTime().Unix() {
				// cache hit
				return nil
			}
			if strings.HasPrefix(path, ".git") {
				// ignore .git
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
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0755)
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
		status.Health = "Updated " + strconv.Itoa(updated) + " files without restart"
	}

	if restart {
		RestartApp("")
		status.Health = "Restarting after updating " + strconv.Itoa(updated) + " files"
	}
	return status
}


func NeedsRestart(path string) bool {
	ignoreRegex := viper.GetString(CONFIG_IGNORE_REGEX)
	if len(ignoreRegex) > 0 {
		match, _ := regexp.MatchString(ignoreRegex, path)
		if match {
			log.Println("Skipping restart for: " + path)
			return false
		}
	}
	restartRegex := viper.GetString(CONFIG_RESTART_REGEX)
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
	appDirsRaw := viper.GetString(CONFIG_BACKEND_DIRS)
	if len(appDirsRaw) > 0 {
		return strings.Fields(appDirsRaw)
	}
	return []string{"./"}
}

func GetBackendEnvironment() []string {
	const portLabel = "PORT"
	var currentEnv = os.Environ()
	var portIndex = -1
	for idx, item := range currentEnv {
		var tokens = strings.Split(item, "=")
		if tokens[0] == portLabel {
			portIndex = idx;
		}
	}
	var portEnv = fmt.Sprintf("%s=%s", portLabel, viper.GetString(CONFIG_BACKEND_PORT))
	if portIndex < 0 {
		return append(currentEnv, portEnv)
	} else {
		currentEnv[portIndex] = portEnv
		return currentEnv
	}
}