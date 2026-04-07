package main

import (
	"log"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

func main() {
	dir := os.Getenv("WEBDAV_DIR")
	if dir == "" {
		log.Fatal("WEBDAV_DIR environment variable is required")
	}

	username := os.Getenv("WEBDAV_USERNAME")
	password := os.Getenv("WEBDAV_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("WEBDAV_USERNAME and WEBDAV_PASSWORD environment variables are required")
	}

	handler := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(dir),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("ERROR: %s %s: %v", r.Method, r.URL.Path, err)
			}
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})

	addr := ":8080"
	log.Printf("Serving WebDAV on %s from directory %s", addr, dir)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
