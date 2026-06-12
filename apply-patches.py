#!/usr/bin/env python3
import os

print("🚀 Aplicando patches no FinOps Platform...")
print("")

def read_file(path):
    try:
        with open(path, 'r') as f:
            return f.read()
    except FileNotFoundError:
        return None

def write_file(path, content):
    with open(path, 'w') as f:
        f.write(content)

# 1. api-gateway/main.go
print("📦 [1/8] Corrigindo api-gateway/main.go...")
content = read_file("backend/api-gateway/main.go")
if content:
    old = 'return `{\"time\":\"` + param.TimeStamp.Format(time.RFC3339) + `\",\"client\":\"` + param.ClientIP + `\",\"method\":\"` + param.Method + `\",\"path\":\"` + param.Path + `\",\"status\":` + string(rune(param.StatusCode)) + `,\"latency\":\"` + param.Latency.String() + `\"}` + "\n"'
    new = 'return fmt.Sprintf(`{\"time\":\"%s\",\"client\":\"%s\",\"method\":\"%s\",\"path\":\"%s\",\"status\":%d,\"latency\":\"%s\"}`+"\\n",\n param.TimeStamp.Format(time.RFC3339),\n param.ClientIP,\n param.Method,\n param.Path,\n param.StatusCode,\n param.Latency.String())'
    content = content.replace(old, new)
    if '"fmt"' not in content:
        content = content.replace('"log"', '"fmt"\n\t"log"')
    write_file("backend/api-gateway/main.go", content)
    print("   ✅ api-gateway/main.go corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 2. ingestion-service/main.go
print("📦 [2/8] Corrigindo ingestion-service/main.go...")
content = read_file("backend/ingestion-service/main.go")
if content:
    for imp in ['"bufio"', '"compress/gzip"', '"encoding/csv"', '"io"']:
        content = content.replace(imp + '\n', '')
    write_file("backend/ingestion-service/main.go", content)
    print("   ✅ ingestion-service/main.go corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 3. ingestion-service/Dockerfile
print("📦 [3/8] Corrigindo ingestion-service/Dockerfile...")
if os.path.exists("backend/ingestion-service/Dockerfile"):
    df = """FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget curl unzip
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8081
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \\
  CMD wget --spider -q http://localhost:8081/health 2>/dev/null || curl -fsS http://localhost:8081/health
CMD ["./main"]
"""
    write_file("backend/ingestion-service/Dockerfile", df)
    print("   ✅ ingestion-service/Dockerfile corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 4. cost-analytics/main.go
print("📦 [4/8] Corrigindo cost-analytics/main.go...")
content = read_file("backend/cost-analytics/main.go")
if content:
    if 'fmt.' not in content:
        content = content.replace('"fmt"\n', '')
    write_file("backend/cost-analytics/main.go", content)
    print("   ✅ cost-analytics/main.go corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 5. cost-analytics/Dockerfile
print("📦 [5/8] Corrigindo cost-analytics/Dockerfile...")
if os.path.exists("backend/cost-analytics/Dockerfile"):
    df = """FROM golang:1.22-alpine AS builder
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
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \\
  CMD wget --spider -q http://localhost:8082/health 2>/dev/null || curl -fsS http://localhost:8082/health
CMD ["./main"]
"""
    write_file("backend/cost-analytics/Dockerfile", df)
    print("   ✅ cost-analytics/Dockerfile corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 6. alert-manager/main.go - CORRIGIDO PARA MULTILINHA
print("📦 [6/8] Corrigindo alert-manager/main.go...")
content = read_file("backend/alert-manager/main.go")
if content:
    # Substituir a parte problemática (funciona para multilinha ou single line)
    old = 'alert.Message + "\nValue: " + string(rune(int(alert.Value))) + "\nThreshold: " + string(rune(int(alert.Threshold)))'
    new = 'fmt.Sprintf("%s\\nValue: %.2f\\nThreshold: %.2f", alert.Message, alert.Value, alert.Threshold)'
    content = content.replace(old, new)

    if '"strconv"' not in content:
        content = content.replace('"bytes"', '"bytes"\n\t"strconv"')
    if '"fmt"' not in content:
        content = content.replace('"bytes"', '"bytes"\n\t"fmt"')
    write_file("backend/alert-manager/main.go", content)
    print("   ✅ alert-manager/main.go corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 7. alert-manager/Dockerfile
print("📦 [7/8] Corrigindo alert-manager/Dockerfile...")
if os.path.exists("backend/alert-manager/Dockerfile"):
    df = """FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget curl
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8083
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \\
  CMD wget --spider -q http://localhost:8083/health 2>/dev/null || curl -fsS http://localhost:8083/health
CMD ["./main"]
"""
    write_file("backend/alert-manager/Dockerfile", df)
    print("   ✅ alert-manager/Dockerfile corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

# 8. api-gateway/Dockerfile
print("📦 [8/8] Corrigindo api-gateway/Dockerfile...")
if os.path.exists("backend/api-gateway/Dockerfile"):
    df = """FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget curl
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \\
  CMD wget --spider -q http://localhost:8080/health 2>/dev/null || curl -fsS http://localhost:8080/health
CMD ["./main"]
"""
    write_file("backend/api-gateway/Dockerfile", df)
    print("   ✅ api-gateway/Dockerfile corrigido")
else:
    print("   ⚠️  arquivo não encontrado")

print("")
print("================================================================================")
print("✅ PATCHES APLICADOS NOS SERVIÇOS GO!")
print("================================================================================")
print("")
