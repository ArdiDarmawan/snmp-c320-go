# ===== build stage =====
FROM golang:1.23-alpine AS build

WORKDIR /app

# deps for go modules
RUN apk add --no-cache git ca-certificates

# copy go mod first for better caching
COPY go.mod go.sum ./
RUN go mod download

# copy the rest
COPY . .

# build (static-ish)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o zte-c320-snmp-api .

# ===== runtime stage =====
FROM alpine:3.20

WORKDIR /app

# CA certs for any TLS you might use later
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 appuser

# copy binary
COPY --from=build /app/zte-c320-snmp-api /app/zte-c320-snmp-api

# default config path (akan di-mount dari host)
ENV CONFIG_PATH=/app/config.yaml
ENV PORT=8080

USER appuser
EXPOSE 8080

# jalankan binary (main.go kamu baca config.yaml; kalau belum, mount ke /app/config.yaml)
CMD ["/app/zte-c320-snmp-api"]
