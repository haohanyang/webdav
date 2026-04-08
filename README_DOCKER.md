# WebDAV Server

A minimal WebDAV server written in Go. Serves a local directory over WebDAV with optional HTTP Basic Authentication.

## Quick Start

```bash
docker run -p 8080:8080 \
  -e WEBDAV_DIR=/data \
  -e WEBDAV_USERNAME=user \
  -e WEBDAV_PASSWORD=secret \
  -v /path/to/files:/data \
  haohanyang/webdav
```

Without authentication:

```bash
docker run -p 8080:8080 \
  -e WEBDAV_DIR=/data \
  -e WEBDAV_NO_AUTH=1 \
  -v /path/to/files:/data \
  haohanyang/webdav
```

## Environment Variables

| Variable          | Required | Description                                |
| ----------------- | -------- | ------------------------------------------ |
| `WEBDAV_DIR`      | Yes      | Path to the directory to serve             |
| `WEBDAV_USERNAME` | No\*     | Basic auth username                        |
| `WEBDAV_PASSWORD` | No\*     | Basic auth password                        |
| `WEBDAV_NO_AUTH`  | No       | Set to any value to disable authentication |

\* Required unless `WEBDAV_NO_AUTH` is set.

The server listens on port `8080`.

## Docker Compose

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

## Source

[github.com/haohanyang/webdav](https://github.com/haohanyang/webdav)
