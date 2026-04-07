# WebDAV Server

A minimal WebDAV server written in Go. Serves a local directory over WebDAV with HTTP Basic Authentication.

## Features

- WebDAV protocol support (PROPFIND, GET, PUT, DELETE, MKCOL, COPY, MOVE, etc.)
- HTTP Basic Authentication
- Directory browsing via GET/HEAD (redirected to PROPFIND)
- In-memory lock system

## Usage

### Environment Variables

| Variable          | Required | Description                    |
| ----------------- | -------- | ------------------------------ |
| `WEBDAV_DIR`      | Yes      | Path to the directory to serve |
| `WEBDAV_USERNAME` | Yes      | Basic auth username            |
| `WEBDAV_PASSWORD` | Yes      | Basic auth password            |

The server listens on port `8080`.

### Run with Go

```bash
WEBDAV_DIR=/path/to/files WEBDAV_USERNAME=user WEBDAV_PASSWORD=secret go run .
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
