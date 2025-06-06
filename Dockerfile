# Multi-stage Dockerfile para OSDO CLI
# Etapa 1: Build (sin CGO para compatibilidad multiplataforma)
FROM golang:1.21-alpine AS builder

# Instalar dependencias de build
RUN apk add --no-cache git ca-certificates tzdata

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de módulo Go
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar aplicación sin CGO para máxima compatibilidad
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=docker -s -w" \
    -o osdo-cli .

# Etapa 2: Runtime
FROM alpine:latest

# Instalar dependencias runtime
RUN apk --no-cache add ca-certificates curl docker-cli kubectl helm

# Crear usuario no-root
RUN addgroup -g 1001 osdo && \
    adduser -D -u 1001 -G osdo osdo

# Crear directorios necesarios
RUN mkdir -p /app/config /app/data && \
    chown -R osdo:osdo /app

# Copiar binario desde stage de build
COPY --from=builder /app/osdo-cli /usr/local/bin/osdo-cli

# Cambiar a usuario no-root
USER osdo

# Establecer directorio de trabajo
WORKDIR /app

# Exponer puerto (si es necesario para monitoreo)
EXPOSE 8080

# Punto de entrada
ENTRYPOINT ["osdo-cli"]

# Comando por defecto
CMD ["--help"]

# Metadata
LABEL maintainer="OSDO Project" \
      description="OSDO DevSecOps CLI - Herramienta de despliegue multiplataforma" \
      version="1.0.0"
