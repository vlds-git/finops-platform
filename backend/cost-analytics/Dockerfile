FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget curl
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8082
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:8082/health 2>/dev/null || curl -fsS http://localhost:8082/health
CMD ["./main"]
