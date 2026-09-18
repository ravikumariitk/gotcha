FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/gotcha .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg && adduser -D -u 10001 gotcha
USER gotcha
COPY --from=build /out/gotcha /usr/local/bin/gotcha
EXPOSE 8082
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8082/api/health || exit 1
ENTRYPOINT ["/usr/local/bin/gotcha"]
