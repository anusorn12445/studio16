# ---- build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache modules (this project uses only the standard library, so this is quick).
COPY go.mod ./
RUN go mod download

COPY . .
# CGO off -> a fully static binary that runs on a bare Alpine image.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.20
# ffmpeg is needed to pull a still frame out of generated clips for the match report.
RUN apk add --no-cache ffmpeg ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /src/web /app/web

ENV PORT=8080
ENV DATA_DIR=/data
ENV WEB_DIR=/app/web
VOLUME ["/data"]
EXPOSE 8080

CMD ["/app/server"]
