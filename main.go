package main

import (
	"fmt"
	"golang.org/x/net/webdav"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const dirTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="description" content="WebDav Server">
<title>Index of {{.Path}}</title>
<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 min-h-screen text-gray-800">

<div class="max-w-5xl mx-auto px-4 py-10">

  <!-- Header -->
  <div class="mb-6">
    <p class="text-xs font-semibold uppercase tracking-widest text-gray-400 mb-1">File Browser</p>
    <h1 class="text-2xl font-bold text-gray-900 break-all font-mono">{{.Path}}</h1>
  </div>

  <!-- Card -->
  <div class="bg-white rounded-2xl shadow-sm border border-gray-200 overflow-hidden">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-gray-200 bg-gray-50">
          <th class="text-left px-5 py-3 font-semibold text-xs uppercase tracking-wider text-gray-500">Name</th>
          <th class="text-left px-5 py-3 font-semibold text-xs uppercase tracking-wider text-gray-500">MIME Type</th>
          <th class="text-left px-5 py-3 font-semibold text-xs uppercase tracking-wider text-gray-500">Modified</th>
          <th class="text-right px-5 py-3 font-semibold text-xs uppercase tracking-wider text-gray-500">Size</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">

        {{if ne .Path "/"}}
        <tr class="hover:bg-blue-50 transition-colors">
          <td class="px-5 py-3" colspan="4">
            <a href="../" class="flex items-center gap-2 text-gray-500 hover:text-blue-600 font-mono">
              <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7"/></svg>
              ..
            </a>
          </td>
        </tr>
        {{end}}

        {{range .Entries}}
        <tr class="hover:bg-blue-50 transition-colors group">
          <td class="px-5 py-3">
            {{if .IsDir}}
            <a href="{{.Name}}/" class="flex items-center gap-2 font-medium text-blue-700 hover:text-blue-900">
              <svg class="w-4 h-4 shrink-0 text-yellow-400" fill="currentColor" viewBox="0 0 20 20"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"/></svg>
              {{.Name}}
            </a>
            {{else}}
            <a href="{{.Name}}" class="flex items-center gap-2 text-gray-700 hover:text-blue-700">
              <svg class="w-4 h-4 shrink-0 text-gray-400" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"/></svg>
              {{.Name}}
            </a>
            {{end}}
          </td>
          <td class="px-5 py-3 text-gray-400 whitespace-nowrap font-mono text-xs">{{.MimeType}}</td>
          <td class="px-5 py-3 text-gray-400 tabular-nums whitespace-nowrap">{{.ModTime}}</td>
          <td class="px-5 py-3 text-right text-gray-400 tabular-nums whitespace-nowrap">
            {{if .IsDir}}<span class="text-gray-300">&mdash;</span>{{else}}{{.HumanSize}}{{end}}
          </td>
        </tr>
        {{end}}

      </tbody>
    </table>
  </div>

  <p class="mt-4 text-xs text-gray-400 text-right">WebDAV File Server</p>
</div>

</body>
</html>`

var tmpl = template.Must(template.New("dir").Parse(dirTemplate))

type dirEntry struct {
	Name      string
	IsDir     bool
	ModTime   string
	HumanSize string
	MimeType  string
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
			e.MimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(info.Name())))
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
