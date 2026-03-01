FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

ARG VERSION=dev

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o sproxy .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/sproxy /usr/local/bin/sproxy

EXPOSE 1080

ENTRYPOINT ["sproxy"]
