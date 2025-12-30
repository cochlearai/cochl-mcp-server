FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o cochl-mcp-server cmd/cochl-mcp-server/main.go

FROM jrottenberg/ffmpeg:7.1-scratch

WORKDIR /app

COPY --from=builder /build/cochl-mcp-server /app/cochl-mcp-server

ENTRYPOINT ["/app/cochl-mcp-server"]
