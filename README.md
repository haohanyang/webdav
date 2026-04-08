# WebDAV Server

A minimal WebDAV server written in Go. Serves a local directory over WebDAV with optional HTTP Basic Authentication.

## Features

- WebDAV protocol support (PROPFIND, GET, PUT, DELETE, MKCOL, COPY, MOVE, etc.)
- HTTP Basic Authentication
- Directory browsing via GET/HEAD (redirected to PROPFIND)
- In-memory lock system

## Usage

### Environment Variables

| Variable          | Required | Description                                              |
| ----------------- | -------- | -------------------------------------------------------- |
| `WEBDAV_DIR`      | Yes      | Path to the directory to serve                           |
| `WEBDAV_USERNAME` | No*      | Basic auth username                                      |
| `WEBDAV_PASSWORD` | No*      | Basic auth password                                      |
| `WEBDAV_NO_AUTH`  | No       | Set to any value to disable authentication               |

\* Required unless `WEBDAV_NO_AUTH` is set.

The server listens on port `8080`.

### Run with Go

```bash
WEBDAV_DIR=/path/to/files WEBDAV_USERNAME=user WEBDAV_PASSWORD=secret go run .
```

Without authentication:

```bash
WEBDAV_DIR=/path/to/files WEBDAV_NO_AUTH=1 go run .
```

### Run with Docker

```bash
docker run -p 8080:8080 \
  -e WEBDAV_DIR=/data \
  -e WEBDAV_USERNAME=user \
  -e WEBDAV_PASSWORD=secret \
  -v /path/to/files:/data \
  <your-dockerhub-username>/webdav
```

### Run with Docker Compose

```yaml
services:
  webdav:
    image: haohanyang/webdav
    ports:
      - 8080:8080
    environment:
      WEBDAV_DIR: /data
      WEBDAV_USERNAME: admin
      WEBDAV_PASSWORD: admin
      # WEBDAV_NO_AUTH: "true"  # uncomment to disable authentication
    volumes:
      - /path/to/my/data:/data
```

## Building

```bash
go build -o webdav .
```

## Docker

A multi-stage Dockerfile is included. The workflow in [.github/workflows/docker-publish.yml](.github/workflows/docker-publish.yml) publishes the image to Docker Hub on manual trigger with a specified tag.

To build locally:

```bash
docker build -t webdav .
```
