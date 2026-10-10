package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	envDir              = "WEBDAV_DIR"
	envPort             = "WEBDAV_PORT"
	envPrefix           = "WEBDAV_PREFIX"
	envNoAuth           = "WEBDAV_NO_AUTH"
	envUsername         = "WEBDAV_USERNAME"
	envPassword         = "WEBDAV_PASSWORD"
	envOIDCIssuer       = "WEBDAV_OIDC_ISSUER"
	envOIDCClientID     = "WEBDAV_OIDC_CLIENT_ID"
	envOIDCClientSecret = "WEBDAV_OIDC_CLIENT_SECRET"
	envOIDCRedirectURL  = "WEBDAV_OIDC_REDIRECT_URL"
	envEmailWhitelist   = "WEBDAV_EMAIL_WHITELIST"
	envIDWhitelist      = "WEBDAV_ID_WHITELIST"
)

type config struct {
	dir              string
	port             string
	prefixPath       string
	noAuth           bool
	username         string
	password         string
	oidcIssuer       string
	oidcClientID     string
	oidcClientSecret string
	oidcRedirectURL  string
	emailWhitelist   string
	idWhitelist      string
}

func loadConfig() (config, error) {
	cfg := config{
		dir:              os.Getenv(envDir),
		port:             os.Getenv(envPort),
		noAuth:           os.Getenv(envNoAuth) != "",
		username:         os.Getenv(envUsername),
		password:         os.Getenv(envPassword),
		oidcIssuer:       os.Getenv(envOIDCIssuer),
		oidcClientID:     os.Getenv(envOIDCClientID),
		oidcClientSecret: os.Getenv(envOIDCClientSecret),
		oidcRedirectURL:  os.Getenv(envOIDCRedirectURL),
		emailWhitelist:   os.Getenv(envEmailWhitelist),
		idWhitelist:      os.Getenv(envIDWhitelist),
	}

	if cfg.dir == "" {
		return config{}, fmt.Errorf("%s environment variable is required", envDir)
	}

	prefix := strings.TrimSpace(os.Getenv(envPrefix))
	if prefix == "" {
		prefix = "dav"
	}
	cfg.prefixPath = path.Clean("/" + strings.Trim(prefix, "/"))
	if cfg.prefixPath == "/" || cfg.prefixPath == "." {
		return config{}, fmt.Errorf("%s must be a non-empty path (default: dav)", envPrefix)
	}

	return cfg, nil
}
