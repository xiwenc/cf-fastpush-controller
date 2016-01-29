package main

import (
	"log"
	"net/http"

	"github.com/spf13/viper"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/xiwenc/cf-fastpush-controller/lib"
)

var app_cmd string
var listenOn string

func main() {

	viper.SetConfigName("cf-fastpush-controller")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	viper.SetDefault(lib.CONFIG_BIND_ADDRESS, "0.0.0.0")
	viper.SetDefault(lib.CONFIG_PORT, "9000")
	viper.SetDefault(lib.CONFIG_BACKEND_DIRS, "./")
	viper.SetDefault(lib.CONFIG_RESTART_REGEX, "*.py^")
	viper.SetDefault(lib.CONFIG_IGNORE_REGEX, "")
	viper.SetDefault(lib.CONFIG_BACKEND_COMMAND, "python -m http.server")
	viper.SetDefault(lib.CONFIG_BACKEND_PORT, "8080")

	app_cmd = viper.GetString(lib.CONFIG_BACKEND_COMMAND)
	listenOn = viper.GetString(lib.CONFIG_BIND_ADDRESS) + ":" + viper.GetString(lib.CONFIG_PORT)

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
	go lib.RestartApp(app_cmd)
	go lib.ListFiles()
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

	inputFiles := map[string]*lib.FileEntry{}
	err := r.DecodeJsonPayload(&inputFiles)
	if err != nil {
		log.Println(err.Error())
		rest.Error(w, err.Error(), http.StatusBadRequest)
	}
	result := lib.UploadFiles(inputFiles)
	w.WriteJson(result)
}
