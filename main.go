package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/net/webdav"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

{{if .User.Email}}
<header class="sticky top-0 z-10 bg-white border-b border-gray-200 shadow-sm">
  <div class="max-w-5xl mx-auto px-4 h-12 flex items-center justify-end">
    <div class="relative" id="user-menu">
      <button id="avatar-btn" type="button"
              class="flex items-center gap-2 rounded-full focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2">
        {{if .User.Picture}}
        <img src="{{.User.Picture}}" class="w-8 h-8 rounded-full select-none" alt="">
        {{else}}
        <span class="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center text-white text-sm font-semibold select-none">{{.User.Initial}}</span>
        {{end}}
        <svg class="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
      </button>
      <div id="user-dropdown"
           class="hidden absolute right-0 mt-2 w-60 bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden z-20">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center gap-3">
          {{if .User.Picture}}
          <img src="{{.User.Picture}}" class="w-9 h-9 rounded-full shrink-0" alt="">
          {{else}}
          <span class="w-9 h-9 rounded-full bg-blue-500 flex items-center justify-center text-white text-sm font-semibold shrink-0">{{.User.Initial}}</span>
          {{end}}
          <span class="text-xs text-gray-600 truncate">{{.User.Email}}</span>
        </div>
        <a href="/auth/logout"
           class="flex items-center gap-2 px-4 py-2.5 text-sm text-gray-700 hover:bg-gray-50 transition-colors">
          <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9"/></svg>
          Sign out
        </a>
      </div>
    </div>
  </div>
</header>
<script>
(function(){
  var btn = document.getElementById('avatar-btn');
  var dd  = document.getElementById('user-dropdown');
  btn.addEventListener('click', function(e){ e.stopPropagation(); dd.classList.toggle('hidden'); });
  document.addEventListener('click', function(){ dd.classList.add('hidden'); });
})();
</script>
{{end}}

<div class="max-w-5xl mx-auto px-4 py-10">

  <!-- Header -->
  <div class="mb-6 flex items-end justify-between gap-4">
    <div>
      <p class="text-xs font-semibold uppercase tracking-widest text-gray-400 mb-1">File Browser</p>
      <h1 class="text-2xl font-bold text-gray-900 break-all font-mono">{{.Path}}</h1>
    </div>
    <div class="shrink-0">
      <button id="upload-btn"
              class="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 active:bg-blue-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"/></svg>
        Upload
      </button>
      <input id="upload-input" type="file" multiple class="hidden">
    </div>
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
          <th class="w-10 px-3 py-3"></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">

        {{if ne .Path "/"}}
        <tr class="hover:bg-blue-50 transition-colors">
          <td class="px-5 py-3" colspan="5">
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
          <td class="px-3 py-3 text-right">
            <button class="action-btn opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-gray-200 text-gray-400 hover:text-gray-600"
                    data-name="{{.Name}}" data-url="{{.Name}}{{if .IsDir}}/{{end}}" aria-label="Actions">
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path d="M10 6a2 2 0 110-4 2 2 0 010 4zm0 6a2 2 0 110-4 2 2 0 010 4zm0 6a2 2 0 110-4 2 2 0 010 4z"/></svg>
            </button>
          </td>
        </tr>
        {{end}}

      </tbody>
    </table>
  </div>

  <!-- Shared action dropdown -->
  <div id="action-dropdown" class="hidden fixed bg-white rounded-xl shadow-lg border border-gray-200 py-1 z-30 w-36">
    <button id="action-rename"
            class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors">
      <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L6.832 19.82a4.5 4.5 0 01-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 011.13-1.897L16.863 4.487zm0 0L19.5 7.125"/></svg>
      Rename
    </button>
    <button id="action-delete"
            class="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors">
      <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"/></svg>
      Delete
    </button>
  </div>
  <script>
  (function(){
    var dropdown = document.getElementById('action-dropdown');
    var currentName = null, currentUrl = null;

    document.querySelectorAll('.action-btn').forEach(function(btn) {
      btn.addEventListener('click', function(e) {
        e.stopPropagation();
        currentName = btn.dataset.name;
        currentUrl  = btn.dataset.url;
        var rect = btn.getBoundingClientRect();
        dropdown.style.top  = (rect.bottom + window.scrollY + 4) + 'px';
        dropdown.style.left = Math.min(rect.left, window.innerWidth - 160) + 'px';
        dropdown.classList.remove('hidden');
      });
    });

    document.getElementById('action-rename').addEventListener('click', function() {
      dropdown.classList.add('hidden');
      if (!currentUrl) return;
      var newName = prompt('Rename "' + currentName + '" to:', currentName);
      if (!newName || newName === currentName) return;
      var base = window.location.pathname;
      if (!base.endsWith('/')) base += '/';
      var dest = window.location.origin + base + encodeURIComponent(newName);
      fetch(currentUrl, {method: 'MOVE', credentials: 'same-origin', headers: {Destination: dest, Overwrite: 'F'}})
        .then(function(r) {
          if (r.ok) { location.reload(); }
          else { alert('Rename failed: ' + r.status + ' ' + r.statusText); }
        })
        .catch(function(err) { alert('Rename failed: ' + err); });
    });

    document.getElementById('action-delete').addEventListener('click', function() {
      dropdown.classList.add('hidden');
      if (!currentUrl) return;
      if (!confirm('Delete "' + currentName + '"?')) return;
      fetch(currentUrl, {method: 'DELETE', credentials: 'same-origin'})
        .then(function(r) {
          if (r.ok) { location.reload(); }
          else { alert('Delete failed: ' + r.status + ' ' + r.statusText); }
        })
        .catch(function(err) { alert('Delete failed: ' + err); });
    });

    document.addEventListener('click', function() { dropdown.classList.add('hidden'); });
  })();
  </script>
  <script>
  (function(){
    var btn   = document.getElementById('upload-btn');
    var input = document.getElementById('upload-input');
    btn.addEventListener('click', function() { input.click(); });
    input.addEventListener('change', function() {
      var files = Array.from(input.files);
      if (!files.length) return;
      var base = window.location.pathname;
      if (!base.endsWith('/')) base += '/';
      btn.disabled = true;
      var done = 0;
      function label() { btn.textContent = 'Uploading ' + done + '/' + files.length + '…'; }
      label();
      Promise.all(files.map(function(f) {
        return fetch(base + encodeURIComponent(f.name), {method: 'PUT', credentials: 'same-origin', body: f})
          .then(function(r) {
            if (!r.ok) throw new Error('"' + f.name + '" failed: ' + r.status + ' ' + r.statusText);
            done++; label();
          });
      })).then(function() {
        location.reload();
      }).catch(function(err) {
        alert('Upload failed: ' + err);
        btn.disabled = false;
        btn.innerHTML = '<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"/></svg> Upload';
        input.value = '';
      });
    });
  })();
  </script>

  <div class="mt-4 flex items-center justify-end gap-2 text-xs text-gray-400">
    <span>WebDAV File Server</span>
    <a href="https://github.com/haohanyang/webdav" target="_blank" rel="noopener" class="hover:text-gray-600 transition-colors" title="GitHub">
      <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path fill-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844a9.59 9.59 0 012.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0022 12.017C22 6.484 17.522 2 12 2z" clip-rule="evenodd"/></svg>
    </a>
  </div>
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

// templateUser holds the logged-in user data exposed to the HTML template.
type templateUser struct {
	Email   string
	Picture string
	Initial string // uppercase first rune of the email local-part
}

func serveIndex(w http.ResponseWriter, r *http.Request, fsRoot, urlPath string) {
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

	user, _ := r.Context().Value(ctxUser{}).(templateUser)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, struct {
		Path    string
		Entries []dirEntry
		User    templateUser
	}{urlPath, append(dirs, files...), user})
}

// ---- session store ----

const (
	sessionCookieName = "webdav_session"
	sessionDuration   = 24 * time.Hour
)

type sessionData struct {
	Email   string
	Picture string
	Expires time.Time
}

type sessionStore struct {
	mu   sync.RWMutex
	data map[string]sessionData
}

func newSessionStore() *sessionStore {
	return &sessionStore{data: make(map[string]sessionData)}
}

func (s *sessionStore) create(email, picture string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.data[token] = sessionData{Email: email, Picture: picture, Expires: time.Now().Add(sessionDuration)}
	s.mu.Unlock()
	return token
}

func (s *sessionStore) get(token string) (sessionData, bool) {
	s.mu.RLock()
	sd, ok := s.data[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(sd.Expires) {
		return sessionData{}, false
	}
	return sd, true
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.data, token)
	s.mu.Unlock()
}

// ---- request context key ----

type ctxUser struct{}

func emailInitial(email string) string {
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	r, _ := utf8.DecodeRuneInString(local)
	return strings.ToUpper(string(r))
}

// ---- auth modes ----

type authMode int

const (
	authNone  authMode = iota
	authBasic          // basic auth only (WebDAV + browser)
	authOIDC           // Google OIDC only (browser only)
	authBoth           // OIDC for browser, basic for WebDAV clients
)

func (m authMode) String() string {
	switch m {
	case authNone:
		return "none"
	case authBasic:
		return "basic"
	case authOIDC:
		return "oidc"
	case authBoth:
		return "basic+oidc"
	default:
		return "unknown"
	}
}

type appConfig struct {
	mode           authMode
	dir            string
	username       string
	password       string
	oauth2Cfg      *oauth2.Config
	emailWhitelist map[string]bool
	sessions       *sessionStore
}

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (cfg *appConfig) checkBasicAuth(r *http.Request) bool {
	u, p, ok := r.BasicAuth()
	return ok && u == cfg.username && p == cfg.password
}

// checkOIDCSession validates the session cookie. On success it returns the
// enriched request (with user stored in context) and true. On failure it
// writes the redirect or error and returns nil, false.
func (cfg *appConfig) checkOIDCSession(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.SetCookie(w, &http.Cookie{Name: "oauth_next", Value: r.RequestURI, Path: "/", HttpOnly: true, MaxAge: 300})
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return nil, false
	}
	sd, ok := cfg.sessions.get(cookie.Value)
	if !ok {
		http.SetCookie(w, &http.Cookie{Name: "oauth_next", Value: r.RequestURI, Path: "/", HttpOnly: true, MaxAge: 300})
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return nil, false
	}
	if !cfg.emailWhitelist[sd.Email] {
		http.Error(w, "Forbidden: "+sd.Email+" is not on the whitelist", http.StatusForbidden)
		return nil, false
	}
	user := templateUser{Email: sd.Email, Picture: sd.Picture, Initial: emailInitial(sd.Email)}
	return r.WithContext(context.WithValue(r.Context(), ctxUser{}, user)), true
}

func (cfg *appConfig) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch cfg.mode {
		case authNone:
			next(w, r)

		case authBasic:
			if !cfg.checkBasicAuth(r) {
				w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)

		case authOIDC:
			r2, ok := cfg.checkOIDCSession(w, r)
			if !ok {
				return
			}
			next(w, r2)

		case authBoth:
			// WebDAV clients send Basic credentials; browsers going through OIDC don't.
			if _, _, hasBasic := r.BasicAuth(); hasBasic {
				if !cfg.checkBasicAuth(r) {
					w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next(w, r)
				return
			}
			r2, ok := cfg.checkOIDCSession(w, r)
			if !ok {
				return
			}
			next(w, r2)
		}
	}
}

// ---- OIDC handlers ----

func (cfg *appConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomToken(16)
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", HttpOnly: true, MaxAge: 300})
	http.Redirect(w, r, cfg.oauth2Cfg.AuthCodeURL(state), http.StatusFound)
}

type googleUserInfo struct {
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

func (cfg *appConfig) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

	token, err := cfg.oauth2Cfg.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		log.Printf("OIDC token exchange: %v", err)
		return
	}

	client := cfg.oauth2Cfg.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	if !cfg.emailWhitelist[info.Email] {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	sessionToken := cfg.sessions.create(info.Email, info.Picture)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	next := "/"
	if c, err := r.Cookie("oauth_next"); err == nil && strings.HasPrefix(c.Value, "/") {
		next = c.Value
		http.SetCookie(w, &http.Cookie{Name: "oauth_next", MaxAge: -1, Path: "/"})
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (cfg *appConfig) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		cfg.sessions.delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

// ---- main ----

func main() {
	dir := os.Getenv("WEBDAV_DIR")
	if dir == "" {
		log.Fatal("WEBDAV_DIR environment variable is required")
	}

	noAuth := os.Getenv("WEBDAV_NO_AUTH") != ""
	username := os.Getenv("WEBDAV_USERNAME")
	password := os.Getenv("WEBDAV_PASSWORD")
	clientID := os.Getenv("WEBDAV_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("WEBDAV_GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("WEBDAV_GOOGLE_REDIRECT_URL")
	emailWhitelistStr := os.Getenv("WEBDAV_EMAIL_WHITELIST")

	hasBasic := username != "" && password != ""
	hasOIDC := clientID != "" && clientSecret != ""

	cfg := &appConfig{dir: dir}

	switch {
	case noAuth:
		cfg.mode = authNone
	case hasBasic && hasOIDC:
		cfg.mode = authBoth
	case hasOIDC:
		cfg.mode = authOIDC
	case hasBasic:
		cfg.mode = authBasic
	default:
		log.Fatal("no auth mode configured: set WEBDAV_NO_AUTH=true, or basic auth/OIDC credentials")
	}

	if hasBasic {
		cfg.username = username
		cfg.password = password
	}

	if hasOIDC {
		if redirectURL == "" {
			log.Fatal("WEBDAV_GOOGLE_REDIRECT_URL is required when using Google OIDC (e.g. http://localhost:8080/auth/callback)")
		}
		cfg.emailWhitelist = make(map[string]bool)
		for _, e := range strings.Split(emailWhitelistStr, ",") {
			if e = strings.TrimSpace(e); e != "" {
				cfg.emailWhitelist[e] = true
			}
		}
		if len(cfg.emailWhitelist) == 0 {
			log.Fatal("WEBDAV_EMAIL_WHITELIST is required when using Google OIDC (comma-separated emails)")
		}
		cfg.sessions = newSessionStore()
		cfg.oauth2Cfg = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}
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

	if cfg.mode == authOIDC || cfg.mode == authBoth {
		http.HandleFunc("/auth/login", cfg.handleLogin)
		http.HandleFunc("/auth/callback", cfg.handleCallback)
		http.HandleFunc("/auth/logout", cfg.handleLogout)
	}

	http.HandleFunc("/", cfg.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			info, err := os.Stat(filepath.Join(dir, r.URL.Path))
			if err == nil && info.IsDir() {
				if !strings.HasSuffix(r.URL.Path, "/") {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
					return
				}
				if r.Method == http.MethodGet {
					serveIndex(w, r, dir, r.URL.Path)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		webdavHandler.ServeHTTP(w, r)
	}))

	port := os.Getenv("WEBDAV_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Serving WebDAV on %s from %s (auth: %s)", addr, dir, cfg.mode)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
