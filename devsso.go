package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
	"strings"
)

const VERSION = "0.1"

var (
	flcf *string = flag.String("c", "", "Config file")
)

type User struct {
	Login   string `json:"login"`
	Headers []map[string]string
}

type Configuration struct {
	Host    string `json:"host"`
	ProxyTo string `json:"proxyto"`
	Listen  string `json:"listen"`
	Users   []User `json:"users"`
}

func newConfig(path string) (config Configuration, err error) {
	cf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(cf, &config)
	return
}

func login(cf Configuration, cookie *http.Cookie, r *http.Request) {
	for _, user := range cf.Users {
		if user.Login == cookie.Value {
			for _, headers := range user.Headers {
				for key, val := range headers {
					r.Header.Add(key, val) // CanonicalMIMEHeaderKey
				}
			}
		}
	}
}

func proxy(cf Configuration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" && "" != r.FormValue("ssoid") {
			cid := &http.Cookie{Name: "ssoid", Value: r.FormValue("ssoid"), Path: "*", Expires: time.Now().Add(356 * 24 * time.Hour), HttpOnly: true}
			w.Header().Set("Set-Cookie", cid.String())
			login(cf, cid, r)
			log.Printf("%s - %s [%s]", cid.Value, r.RequestURI, r.Header)
		} else {
			cid, _ := r.Cookie("ssoid")
			if cid != nil {
				login(cf, cid, r)
				log.Printf("%s - %s [%s]", cid.Value, r.RequestURI, r.Header)
			}
		}

		r.Host = cf.Host
		u, _ := url.Parse(cf.ProxyTo)
		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.ServeHTTP(w, r)
	})
}

func main() {

	flag.Parse()

	if *flcf == "" {
		log.Fatal("Voir le -h")
	}

	cf, err := newConfig(*flcf)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting...")
	http.Handle("/", proxy(cf))
	log.Printf("listen %s", cf.Listen)
	log.Fatal(http.ListenAndServe(cf.Listen, nil))

}
