package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	zoidc "github.com/zitadel/oidc/v3/pkg/oidc"
	"golang.org/x/net/webdav"
)

//go:embed template.html
var dirTemplate string

//go:embed login.html
var loginPage string

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
	Name    string
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
	Id      string
	Email   string
	Name    string
	Picture string
	IDToken string
	Expires time.Time
}

type sessionStore struct {
	mu   sync.RWMutex
	data map[string]sessionData
}

func newSessionStore() *sessionStore {
	return &sessionStore{data: make(map[string]sessionData)}
}

func (s *sessionStore) create(id, email, name, picture, idToken string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.data[token] = sessionData{Id: id, Email: email, Name: name, Picture: picture, IDToken: idToken, Expires: time.Now().Add(sessionDuration)}
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
	authOIDC           // OIDC only (browser only)
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
	baseURL        string // scheme+host, e.g. "https://example.com" — used for OIDC redirect URIs
	oidcProvider   rp.RelyingParty
	emailWhitelist map[string]bool
	idWhitelist    map[string]bool
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
		http.Redirect(w, r, "/auth/sign-in", http.StatusFound)
		return nil, false
	}
	sd, ok := cfg.sessions.get(cookie.Value)
	if !ok {
		http.SetCookie(w, &http.Cookie{Name: "oauth_next", Value: r.RequestURI, Path: "/", HttpOnly: true, MaxAge: 300})
		http.Redirect(w, r, "/auth/sign-in", http.StatusFound)
		return nil, false
	}

	if !cfg.idWhitelist[sd.Id] && !cfg.emailWhitelist[sd.Email] {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return nil, false
	}

	user := templateUser{Email: sd.Email, Name: sd.Name, Picture: sd.Picture, Initial: emailInitial(sd.Email)}
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
	http.Redirect(w, r, rp.AuthURL(state, cfg.oidcProvider), http.StatusFound)
}

func (cfg *appConfig) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

	tokens, err := rp.CodeExchange[*zoidc.IDTokenClaims](r.Context(), r.URL.Query().Get("code"), cfg.oidcProvider)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		log.Printf("OIDC token exchange: %v", err)
		return
	}
	if tokens.IDTokenClaims == nil {
		http.Error(w, "ID token missing from token response", http.StatusInternalServerError)
		return
	}

	userInfo, err := rp.Userinfo[*zoidc.UserInfo](r.Context(), tokens.AccessToken, tokens.TokenType, tokens.IDTokenClaims.GetSubject(), cfg.oidcProvider)
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		log.Printf("OIDC userinfo: %v", err)
		return
	}

	if !userInfo.EmailVerified {
		http.Error(w, "Unauthorized: email not verified", http.StatusForbidden)
		return
	}

	sub := userInfo.Subject
	if !cfg.emailWhitelist[userInfo.Email] && !cfg.idWhitelist[sub] {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	sessionToken := cfg.sessions.create(sub, userInfo.Email, userInfo.Name, userInfo.Picture, tokens.IDToken)
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var idToken string
	if c, err := r.Cookie(sessionCookieName); err == nil {
		if sd, ok := cfg.sessions.get(c.Value); ok {
			idToken = sd.IDToken
		}
		cfg.sessions.delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, Path: "/"})

	// Attempt RP-initiated logout so the provider session is also cleared.
	// Falls back to a local signed-out page for providers without end_session_endpoint.
	postLogoutURI := cfg.baseURL + "/auth/sign-in"
	if logoutURL, err := rp.EndSession(r.Context(), cfg.oidcProvider, idToken, postLogoutURI, "", "", nil); err == nil {
		http.Redirect(w, r, logoutURL.String(), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/auth/sign-in", http.StatusFound)
}

func handleSignIn(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, loginPage)
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
	clientID := os.Getenv("WEBDAV_OIDC_CLIENT_ID")
	clientSecret := os.Getenv("WEBDAV_OIDC_CLIENT_SECRET")
	redirectURL := os.Getenv("WEBDAV_OIDC_REDIRECT_URL")
	issuer := os.Getenv("WEBDAV_OIDC_ISSUER")
	emailWhitelistStr := os.Getenv("WEBDAV_EMAIL_WHITELIST")
	idWhitelistStr := os.Getenv("WEBDAV_ID_WHITELIST")

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
		if issuer == "" {
			log.Fatal("WEBDAV_OIDC_ISSUER is required when using OIDC (e.g. https://accounts.google.com)")
		}
		if redirectURL == "" {
			log.Fatal("WEBDAV_OIDC_REDIRECT_URL is required when using OIDC (e.g. http://localhost:8080/auth/callback)")
		}
		if u, err := url.Parse(redirectURL); err == nil {
			cfg.baseURL = u.Scheme + "://" + u.Host
		}

		cfg.emailWhitelist = make(map[string]bool)
		for e := range strings.SplitSeq(emailWhitelistStr, ",") {
			if e = strings.TrimSpace(e); e != "" {
				cfg.emailWhitelist[e] = true
			}
		}
		cfg.idWhitelist = make(map[string]bool)
		for id := range strings.SplitSeq(idWhitelistStr, ",") {
			if id = strings.TrimSpace(id); id != "" {
				cfg.idWhitelist[id] = true
			}
		}
		if len(cfg.emailWhitelist) == 0 && len(cfg.idWhitelist) == 0 {
			log.Fatal("WEBDAV_EMAIL_WHITELIST or WEBDAV_ID_WHITELIST is required when using OIDC")
		}

		provider, err := rp.NewRelyingPartyOIDC(
			context.Background(),
			issuer,
			clientID,
			clientSecret,
			redirectURL,
			[]string{"openid", "email", "profile"},
		)
		if err != nil {
			log.Fatalf("Failed to initialize OIDC provider: %v", err)
		}
		cfg.oidcProvider = provider
		cfg.sessions = newSessionStore()
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
		http.HandleFunc("/auth/sign-in", handleSignIn)
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
