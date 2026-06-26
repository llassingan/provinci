# Stage 1: Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /api ./cmd/api

# Stage 2: Runtime with terraform
FROM hashicorp/terraform:1.9 AS terraform

FROM alpine:3.20
RUN apk add --no-cache ca-certificates openssl
# Copy terraform binary from terraform image
COPY --from=terraform /bin/terraform /usr/local/bin/terraform
WORKDIR /app
COPY --from=builder /api .
COPY ansible/ ansible/
COPY terraform/ terraform/
RUN mkdir -p /app/data
EXPOSE 10000
CMD ["./api"]
