package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"golang.org/x/net/webdav"
)

const dirTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Index of {{.Path}}</title>
<style>
  body { font-family: sans-serif; max-width: 960px; margin: 40px auto; padding: 0 20px; color: #333; }
  h1 { border-bottom: 1px solid #ddd; padding-bottom: 10px; word-break: break-all; }
  table { width: 100%; border-collapse: collapse; }
  th { text-align: left; padding: 8px 12px; border-bottom: 2px solid #ddd; color: #666;
       font-size: 0.85em; text-transform: uppercase; letter-spacing: 0.05em; }
  td { padding: 8px 12px; border-bottom: 1px solid #eee; }
  tr:hover td { background: #f9f9f9; }
  a { color: #0066cc; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .dir a { font-weight: 600; }
  .size { text-align: right; color: #888; font-variant-numeric: tabular-nums; }
  .mtime { color: #888; white-space: nowrap; }
</style>
</head>
<body>
<h1>Index of {{.Path}}</h1>
<table>
  <tr><th>Name</th><th>Modified</th><th class="size">Size</th></tr>
  {{if ne .Path "/"}}
  <tr><td><a href="../">../</a></td><td></td><td></td></tr>
  {{end}}
  {{range .Entries}}
  <tr class="{{if .IsDir}}dir{{end}}">
    <td>{{if .IsDir}}<a href="{{.Name}}/">{{.Name}}/</a>{{else}}<a href="{{.Name}}">{{.Name}}</a>{{end}}</td>
    <td class="mtime">{{.ModTime}}</td>
    <td class="size">{{if .IsDir}}&mdash;{{else}}{{.HumanSize}}{{end}}</td>
  </tr>
  {{end}}
</table>
</body>
</html>`

var tmpl = template.Must(template.New("dir").Parse(dirTemplate))

type dirEntry struct {
	Name      string
	IsDir     bool
	ModTime   string
	HumanSize string
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func serveIndex(w http.ResponseWriter, fsRoot, urlPath string) {
	fsPath := filepath.Join(fsRoot, filepath.FromSlash(urlPath))
	infos, err := os.ReadDir(fsPath)
	if err != nil {
		http.Error(w, "cannot read directory", http.StatusInternalServerError)
		return
	}

	var dirs, files []dirEntry
	for _, info := range infos {
		fi, err := info.Info()
		if err != nil {
			continue
		}
		e := dirEntry{
			Name:    info.Name(),
			IsDir:   info.IsDir(),
			ModTime: fi.ModTime().UTC().Format("2006-01-02 15:04 UTC"),
		}
		if !info.IsDir() {
			e.HumanSize = humanSize(fi.Size())
		}
		if info.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, struct {
		Path    string
		Entries []dirEntry
	}{urlPath, append(dirs, files...)})
}

func main() {
	dir := os.Getenv("WEBDAV_DIR")
	if dir == "" {
		log.Fatal("WEBDAV_DIR environment variable is required")
	}

	noAuth := os.Getenv("WEBDAV_NO_AUTH") != ""

	username := os.Getenv("WEBDAV_USERNAME")
	password := os.Getenv("WEBDAV_PASSWORD")
	if !noAuth && (username == "" || password == "") {
		log.Fatal("WEBDAV_USERNAME and WEBDAV_PASSWORD environment variables are required")
	}

	webdavHandler := &webdav.Handler{
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
		if !noAuth {
			u, p, ok := r.BasicAuth()
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			info, err := os.Stat(filepath.Join(dir, r.URL.Path))
			if err == nil && info.IsDir() {
				if !strings.HasSuffix(r.URL.Path, "/") {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
					return
				}
				if r.Method == http.MethodGet {
					serveIndex(w, dir, r.URL.Path)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		webdavHandler.ServeHTTP(w, r)
	})

	addr := ":8080"
	log.Printf("Serving WebDAV on %s from directory %s", addr, dir)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
