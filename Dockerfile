FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/proxy ./cmd/proxy

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/proxy /proxy
EXPOSE 8080
ENTRYPOINT ["/proxy"]