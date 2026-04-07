FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o webdav .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/webdav .
EXPOSE 8080
CMD ["./webdav"]
