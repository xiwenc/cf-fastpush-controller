package main

import (
	"github.com/xiwenc/go-berlin/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/xiwenc/go-berlin/Godeps/_workspace/src/github.com/matryer/runner"
	"github.com/xiwenc/go-berlin/utils"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var app_cmd string
var listenOn = "localhost:9000"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: " + os.Args[0] + " backend_command [bind_address:port]")
	}
	app_cmd = os.Args[1]
	if len(os.Args) >= 3 {
		listenOn = os.Args[2]
	}
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	router, err := rest.MakeRouter(
		rest.Get("/files", GetFiles),
		rest.Post("/kick", RestartApp),
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	RestartAppInternal()
	log.Fatal(http.ListenAndServe(listenOn, api.MakeHandler()))
}

type FileEntry struct {
	Name     string
	Checksum string
}

var store = []*FileEntry{}

var lock = sync.RWMutex{}
var task *runner.Task
var cmd *exec.Cmd

func GetFiles(w rest.ResponseWriter, r *rest.Request) {
	dir := "./"
	lock.RLock()
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		fileEntry := FileEntry{}
		fileEntry.Name = path
		checksum, _ := utils.ChecksumsForFile(path)
		fileEntry.Checksum = checksum.SHA256
		store = append(store, &fileEntry)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	lock.RUnlock()

	w.WriteJson(store)
}

func RestartAppInternal() {
	lock.RLock()
	parts := strings.Fields(app_cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

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
}

func RestartApp(w rest.ResponseWriter, r *rest.Request) {
	RestartAppInternal()
	fileEntry := FileEntry{}
	w.WriteJson(fileEntry)
}
