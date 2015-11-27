package main

import (
	"github.com/ant0ine/go-json-rest/rest"
	"log"
	"net/http"
	"sync"
	"path/filepath"
	"os"
	"./utils"
	"strings"
	"os/exec"
	"github.com/matryer/runner"
)

var app_cmd = "python -m http.server"

func main() {
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
	log.Fatal(http.ListenAndServe(":8080", api.MakeHandler()))
}

type FileEntry struct {
	Name     string
	Checksum string
}

var store = []*FileEntry{}

var lock = sync.RWMutex{}
var task *runner.Task

func GetFiles(w rest.ResponseWriter, r *rest.Request) {
	dir := "./";
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

func RestartApp(w rest.ResponseWriter, r *rest.Request) {
	lock.RLock()
	parts := strings.Fields(app_cmd)
    head := parts[0]
    parts = parts[1:len(parts)]
	cmd := exec.Command(head, parts...)

	if task != nil {
		task.Stop()
	}

	task = runner.Go(func(shouldStop runner.S) error {
		for {
			cmd.Start()
			if shouldStop() {
				cmd.Process.Kill()
				break
			}
		}
		return nil
	})

	lock.RUnlock()
	fileEntry := FileEntry{}
	w.WriteJson(fileEntry)
}
