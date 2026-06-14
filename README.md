# WebDAV Server

A minimal WebDAV server.

![](static/screenshot.png)

## Features

- WebDAV protocol support (PROPFIND, GET, PUT, DELETE, MKCOL, COPY, MOVE, etc.)
- Directory browsing in the browser
- Basic auth and OIDC (any standards-compliant provider) for authentication

## Environment Variables

| Variable                    | Description                                                           |
| --------------------------- | --------------------------------------------------------------------- |
| `WEBDAV_DIR`                | **(Required)** Path to the directory to serve                         |
| `WEBDAV_PORT`               | Port to listen on (default: `8080`)                                   |
| `WEBDAV_NO_AUTH`            | Set to any value to disable authentication                            |
| `WEBDAV_USERNAME`           | Basic auth username                                                   |
| `WEBDAV_PASSWORD`           | Basic auth password                                                   |
| `WEBDAV_OIDC_ISSUER`        | OIDC issuer URL (e.g. `https://accounts.google.com`)                  |
| `WEBDAV_OIDC_CLIENT_ID`     | OIDC client ID                                                        |
| `WEBDAV_OIDC_CLIENT_SECRET` | OIDC client secret                                                    |
| `WEBDAV_OIDC_REDIRECT_URL`  | OIDC redirect URL (e.g. `http://localhost:8080/auth/callback`)        |
| `WEBDAV_EMAIL_WHITELIST`    | Comma-separated list of allowed emails (matched against OIDC `email`) |
| `WEBDAV_ID_WHITELIST`       | Comma-separated list of allowed subject IDs (matched against OIDC `sub`) |

## Authentication Modes

| Mode          | Variables set                                      | Who can access                                                    |
| ------------- | -------------------------------------------------- | ----------------------------------------------------------------- |
| No auth       | `WEBDAV_NO_AUTH`                                   | Everyone                                                          |
| Basic auth    | `WEBDAV_USERNAME` + `WEBDAV_PASSWORD`              | WebDAV clients and browsers (native dialog)                       |
| OIDC          | OIDC vars + whitelist                              | Browsers only (via SSO login page)                                |
| Basic + OIDC  | OIDC vars + whitelist + `WEBDAV_USERNAME` + `WEBDAV_PASSWORD` | Browsers via OIDC or password form; WebDAV clients via basic auth |

## Usage

### No authentication

```bash
export WEBDAV_DIR=/path/to/files
export WEBDAV_NO_AUTH=true

go run .
```

### Basic auth

```bash
export WEBDAV_DIR=/path/to/files
export WEBDAV_USERNAME=user
export WEBDAV_PASSWORD=secret

go run .
```

### OIDC

Uses OIDC discovery, so any standards-compliant provider works (Google, Microsoft Entra, Keycloak, Zitadel, etc.).

**Google:**

Create an OAuth 2.0 Client ID in the [Google Cloud Console](https://console.cloud.google.com/apis/credentials), add your redirect URL as an authorised redirect URI, then:

```bash
export WEBDAV_DIR=/path/to/files
export WEBDAV_OIDC_ISSUER=https://accounts.google.com
export WEBDAV_OIDC_CLIENT_ID=xxx.apps.googleusercontent.com
export WEBDAV_OIDC_CLIENT_SECRET=xxx
export WEBDAV_OIDC_REDIRECT_URL=http://localhost:8080/auth/callback
export WEBDAV_EMAIL_WHITELIST=alice@gmail.com,bob@gmail.com

go run .
```

**Microsoft Entra ID:**

```bash
export WEBDAV_OIDC_ISSUER=https://login.microsoftonline.com/{tenant}/v2.0
export WEBDAV_OIDC_CLIENT_ID=xxx
export WEBDAV_OIDC_CLIENT_SECRET=xxx
export WEBDAV_OIDC_REDIRECT_URL=http://localhost:8080/auth/callback
export WEBDAV_EMAIL_WHITELIST=alice@example.com
```

At least one of `WEBDAV_EMAIL_WHITELIST` or `WEBDAV_ID_WHITELIST` is required when using OIDC.

### Basic auth + OIDC

Set all of the above variables together. Browsers are authenticated via OIDC SSO or a username/password form; WebDAV clients use basic auth credentials directly.

### Docker Compose

```yaml
services:
  webdav:
    image: haohanyang/webdav
    ports:
      - "8080:8080"
    environment:
      WEBDAV_DIR: /data
      WEBDAV_USERNAME: admin
      WEBDAV_PASSWORD: secret
      # OIDC (optional)
      # WEBDAV_OIDC_ISSUER: https://accounts.google.com
      # WEBDAV_OIDC_CLIENT_ID: xxx.apps.googleusercontent.com
      # WEBDAV_OIDC_CLIENT_SECRET: xxx
      # WEBDAV_OIDC_REDIRECT_URL: http://your-host:8080/auth/callback
      # WEBDAV_EMAIL_WHITELIST: alice@gmail.com,bob@gmail.com
    volumes:
      - /path/to/my/data:/data
```
