FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates wget curl
RUN adduser -D -u 1000 app
WORKDIR /home/app
COPY --from=builder /app/main .
RUN chown -R app:app /home/app
USER app
EXPOSE 8082
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost:8082/health 2>/dev/null || curl -fsS http://localhost:8082/health
CMD ["./main"]
