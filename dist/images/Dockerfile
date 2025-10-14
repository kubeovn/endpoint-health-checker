FROM docker-mirrors.alauda.cn/library/golang:1.24.6-alpine AS builder

WORKDIR /workspace
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o endpoint_health_checker ./main.go
RUN apk add --no-cache libcap && setcap cap_net_raw+ep endpoint_health_checker

FROM scratch
COPY --from=builder /workspace/endpoint_health_checker /endpoint_health_checker
ENTRYPOINT ["/endpoint_health_checker"] 