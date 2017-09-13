package main

import (
	"log"
	"net/url"
	"net/http"
	"net/http/httputil"
	"encoding/json"

	"github.com/spf13/viper"
	"github.com/xiwenc/cf-fastpush-controller/lib"
	"os"
	"strings"
)

type VCAP_APPLICATION struct {
	ApplicationID      string `json:"application_id"`
}
var appCmd string
var listenOn string
var backendOn string

func main() {

	viper.SetConfigName("cf-fastpush-controller")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath(".")
	viper.ReadInConfig()
	viper.AutomaticEnv()

	viper.SetDefault(lib.CONFIG_BIND_ADDRESS, "0.0.0.0")
	viper.SetDefault(lib.CONFIG_PORT, "9000")
	viper.SetDefault(lib.CONFIG_BACKEND_DIRS, "./")
	viper.SetDefault(lib.CONFIG_RESTART_REGEX, "^*.py$")
	viper.SetDefault(lib.CONFIG_IGNORE_REGEX, "")
	viper.SetDefault(lib.CONFIG_BACKEND_COMMAND, "python -m http.server")
	viper.SetDefault(lib.CONFIG_BACKEND_PORT, "8080")
	viper.SetDefault(lib.CONFIG_BASE_PATH, "/_fastpush")

	appCmd = viper.GetString(lib.CONFIG_BACKEND_COMMAND)
	listenOn = viper.GetString(lib.CONFIG_BIND_ADDRESS) + ":" + viper.GetString(lib.CONFIG_PORT)
	backendOn = "127.0.0.1:" + viper.GetString(lib.CONFIG_BACKEND_PORT)
	basePath := viper.GetString(lib.CONFIG_BASE_PATH)
	localAuthToken := GetLocalToken();

	log.Println("Controller listening to: " + listenOn)

	http.HandleFunc(basePath + "/files", func(w http.ResponseWriter, r *http.Request) {
		SetJsonContentType(w)
		if !IsAuthenticated(r, localAuthToken) {
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}
		if r.Method == "GET" {
			ListFiles(w, r)
		} else if r.Method == "PUT" {
			UploadFiles(w, r)
		} else {
			http.Error(w, "Invalid request method.", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc(basePath + "/restart", func(w http.ResponseWriter, r *http.Request) {
		SetJsonContentType(w)
		if !IsAuthenticated(r, localAuthToken) {
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}
		if r.Method == "POST" {
			RestartApp(w, r)
		} else {
			http.Error(w, "Invalid request method.", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc(basePath + "/status", func(w http.ResponseWriter, r *http.Request) {
		SetJsonContentType(w)
		if !IsAuthenticated(r, localAuthToken) {
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}
		if r.Method == "GET" {
			GetStatus(w, r)
		} else {
			http.Error(w, "Invalid request method.", http.StatusMethodNotAllowed)
		}
	})
	reverseProxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   backendOn,
	})
	http.HandleFunc("/", ReverseProxyHandler(reverseProxy))

	go lib.RestartApp(appCmd)
	go lib.ListFiles()
	http.ListenAndServe(listenOn, nil)
}

func SetJsonContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func IsAuthenticated(r *http.Request, localToken string) bool {
	incomingToken := r.Header.Get("x-auth-token")
	if strings.Compare(localToken, "") != 0 && strings.Compare(localToken, incomingToken) == 0 {
		return true
	} else {
		return false
	}
}

func GetLocalToken() string {
	envs := os.Environ()
	var vcap VCAP_APPLICATION

	for _, env := range envs {
		tokens := strings.SplitN(env, "=", 2)
		if strings.Compare(tokens[0], "VCAP_APPLICATION") == 0 {
			data := []byte(tokens[1])
			if err := json.Unmarshal(data, &vcap); err != nil {
				panic(err)
			} else {
				token := vcap.ApplicationID
				log.Println("Authentication token: " + token)
				return token
			}
		}
	}
	log.Println("Unable to determine local Authentication Token")
	return ""
}


func ReverseProxyHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		p.ServeHTTP(w, r)
	}
}

func ListFiles(w http.ResponseWriter, r *http.Request) {
	files := lib.ListFiles()
	json.NewEncoder(w).Encode(files)
}

func RestartApp(w http.ResponseWriter, r *http.Request) {
	result := lib.RestartApp(appCmd)
	json.NewEncoder(w).Encode(result)
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	result := lib.GetStatus()
	json.NewEncoder(w).Encode(result)
}

func UploadFiles(w http.ResponseWriter, r *http.Request) {
	inputFiles := map[string]*lib.FileEntry{}
	err := json.NewDecoder(r.Body).Decode(&inputFiles)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	result := lib.UploadFiles(inputFiles)
	json.NewEncoder(w).Encode(result)
}
