package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/xiwenc/cf-fastpush-controller/lib"
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
		rest.Get("/files", ListFiles),
		rest.Post("/restart", RestartApp),
		rest.Get("/status", GetStatus),
		rest.Put("/files", UploadFiles),
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	lib.RestartApp(app_cmd)
	log.Fatal(http.ListenAndServe(listenOn, api.MakeHandler()))
}


func ListFiles(w rest.ResponseWriter, r *rest.Request) {
	files := lib.ListFiles()
	w.WriteJson(files)
}

func RestartApp(w rest.ResponseWriter, r *rest.Request) {
	result := lib.RestartApp(app_cmd)
	w.WriteJson(result)
}

func GetStatus(w rest.ResponseWriter, r *rest.Request) {
	result := lib.GetStatus()
	w.WriteJson(result)
}

func UploadFiles(w rest.ResponseWriter, r *rest.Request) {
	inputFiles := []lib.FileEntry{}
	r.DecodeJsonPayload(&inputFiles)
	result := lib.UploadFiles(inputFiles)
	w.WriteJson(result)
}
