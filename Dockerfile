# 阶段一：编译
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o aegis-gateway ./cmd/api

# 阶段二：运行
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/aegis-gateway .
COPY --from=builder /app/scripts/lua ./scripts/lua
EXPOSE 8080
CMD ["./aegis-gateway"]
