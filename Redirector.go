package main

import (
	"fmt"
	auth "github.com/abbot/go-http-auth"
	"github.com/go-fsnotify/fsnotify"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const Config = "/tmp/Redirector.txt"
const Log = "/tmp/Redirector.log"
const Form = "./Redirector.form"

var Redirects = make(map[string]string)

func ReadConfig() {
	// read the configuration into global Redirects map
	content, err := ioutil.ReadFile(Config)
	if err != nil {
		log.Println("Cannot read config ", Config)
		f, err := os.OpenFile(Config, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Println("Cannot create config %s", Config)
		} else {
			log.Println("Created empy config ", Config)
			defer f.Close()
		}
	}

	// delete current items
	for k := range Redirects {
		delete(Redirects, k)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {

		//if !strings.HasPrefix(line, "#") {
		s := strings.Fields(line)

		if len(s) == 2 {

			Redirects[s[0]] = s[1]
		}

		//}
	}

	for key, value := range Redirects {
		log.Printf("%v -> %v", key, value)
	}

}

func WatchConfig() {
	// watch the configfile and re-read on changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				//log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					//log.Println("============== MODIFIED %s================", event.Name)
					ReadConfig()
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(Config)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func Secret(user, realm string) string {
	if user == "demo" {
		// password is "hello"
		return "$1$dlPL2MqE$oQmn16q49SqdmhenQuNgs1"
	}
	return ""
}

func edit(w http.ResponseWriter, r *auth.AuthenticatedRequest) {

	host := r.Host
	user_agent := r.UserAgent()
	remote_addr := r.RemoteAddr

	log.Printf("EDIT: %s %s [%s]", host, remote_addr, user_agent)

	if r.Method == "POST" {
		r.ParseForm()
		data := []byte(r.Form.Get("redirects"))
		ioutil.WriteFile(Config, data, 0777)
		// Cookie needs to be accessible in Javascript, therefore not HttpOnly
		cookie := &http.Cookie{Name: "flash_message", Value: "Changes commited", Expires: time.Now().Add(1 * time.Hour), HttpOnly: false}
		http.SetCookie(w, cookie)
	}

	content, err := ioutil.ReadFile(Config)
	if err != nil {
		log.Println("Cannot read ", Config)
		return
	}

	// send a sorted list to browser
	s := strings.Split(string(content), "\n")
	sort.Strings(s)
	content = []byte(strings.Join(s, "\n"))

	w.Header().Add("Content-Type", "text/html")
	w.Header().Add("Server", "WEBtic Redirector Service")
	template, t := ioutil.ReadFile("./redirector.form")
	if t == nil {

		a := strings.Replace(string(template), "{{content}}", string(content), 1)
		fmt.Fprintf(w, "%v", a)

	} else {
		fmt.Fprintf(w, "Cannot read %s", Form)
	}

}

func redirect(w http.ResponseWriter, r *http.Request) {

	host := r.Host
	url := r.URL.Path
	complete := host + url

	referer := r.Referer()
	user_agent := r.UserAgent()
	remote_addr := r.RemoteAddr

	// try redirect for complete path first
	if val, ok := Redirects[complete]; ok {

		log.Printf("complete: %s -> %s for %s [%s] [%s]", complete, val, remote_addr, user_agent, referer)
		http.Redirect(w, r, val, 301)
		return

	}

	// otherwise we try just the hostname
	if val, ok := Redirects[host]; ok {
		log.Printf("host: %s -> %s for %s [%s] [%s]", host, val, remote_addr, user_agent, referer)
		http.Redirect(w, r, val, 301)
		return

	}

	log.Printf("FAIL: %s -> %s for %s [%s] [%s]", host, "not_defined", remote_addr, user_agent, referer)

	http.Redirect(w, r, "http://www.google.com/", 301)
}

func main() {
	log.Println("START")

	f, err := os.OpenFile(Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening logfile: %v", err)
	}
	defer f.Close()

	log.Println("** read config")
	ReadConfig()

	log.Println("** watch config")
	go WatchConfig()

	authenticator := auth.NewBasicAuthenticator("WEBtic Redirector Service", Secret)

	http.HandleFunc("/", redirect)
	http.HandleFunc("/edit", authenticator.Wrap(edit))

	//log.SetOutput(f)

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
