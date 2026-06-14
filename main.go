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
	userInfoURL    string
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
	resp, err := client.Get(cfg.userInfoURL)
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var info struct {
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
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

// ---- OIDC discovery ----

type oidcDiscovery struct {
	AuthURL     string `json:"authorization_endpoint"`
	TokenURL    string `json:"token_endpoint"`
	UserInfoURL string `json:"userinfo_endpoint"`
}

func fetchOIDCDiscovery(issuer string) (*oidcDiscovery, error) {
	url := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned %s", resp.Status)
	}
	var d oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if d.AuthURL == "" || d.TokenURL == "" || d.UserInfoURL == "" {
		return nil, fmt.Errorf("discovery document missing required endpoints")
	}
	return &d, nil
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
	issuerURL := os.Getenv("WEBDAV_OIDC_ISSUER")
	clientID := os.Getenv("WEBDAV_OIDC_CLIENT_ID")
	clientSecret := os.Getenv("WEBDAV_OIDC_CLIENT_SECRET")
	redirectURL := os.Getenv("WEBDAV_OIDC_REDIRECT_URL")
	emailWhitelistStr := os.Getenv("WEBDAV_EMAIL_WHITELIST")

	hasBasic := username != "" && password != ""
	hasOIDC := issuerURL != "" && clientID != "" && clientSecret != ""

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
		log.Fatal("no auth mode configured: set WEBDAV_NO_AUTH=true, WEBDAV_USERNAME+WEBDAV_PASSWORD, or WEBDAV_OIDC_ISSUER+WEBDAV_OIDC_CLIENT_ID+WEBDAV_OIDC_CLIENT_SECRET")
	}

	if hasBasic {
		cfg.username = username
		cfg.password = password
	}

	if hasOIDC {
		if redirectURL == "" {
			log.Fatal("WEBDAV_OIDC_REDIRECT_URL is required when using OIDC (e.g. http://localhost:8080/auth/callback)")
		}
		cfg.emailWhitelist = make(map[string]bool)
		for _, e := range strings.Split(emailWhitelistStr, ",") {
			if e = strings.TrimSpace(e); e != "" {
				cfg.emailWhitelist[e] = true
			}
		}
		if len(cfg.emailWhitelist) == 0 {
			log.Fatal("WEBDAV_EMAIL_WHITELIST is required when using OIDC (comma-separated emails)")
		}

		// Fetch OIDC discovery document to get endpoints.
		discovery, err := fetchOIDCDiscovery(issuerURL)
		if err != nil {
			log.Fatalf("OIDC discovery failed for %s: %v", issuerURL, err)
		}
		log.Printf("OIDC provider: issuer=%s", issuerURL)

		cfg.userInfoURL = discovery.UserInfoURL
		cfg.sessions = newSessionStore()
		cfg.oauth2Cfg = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  discovery.AuthURL,
				TokenURL: discovery.TokenURL,
			},
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
