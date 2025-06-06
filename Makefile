# Makefile para OSDO CLI
.PHONY: build build-gui build-cli clean test install help

# Variables
BINARY_NAME=osdo
VERSION?=1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -s -w"

# Targets por defecto
help: ## Mostrar esta ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Compilación CLI pura (sin CGO)
build-cli: ## Compilar versión CLI sin GUI (recomendado para producción)
	@echo "Compilando versión CLI sin GUI..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Binarios CLI generados en $(BUILD_DIR)/"

# Compilación con GUI (requiere CGO)
build-gui: ## Compilar versión con GUI (requiere CGO y dependencias del sistema)
	@echo "Compilando versión con GUI..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -tags gui -o $(BUILD_DIR)/$(BINARY_NAME)-gui .
	@echo "Binario GUI generado: $(BUILD_DIR)/$(BINARY_NAME)-gui"

# Compilación local para desarrollo
build: build-cli ## Compilar versión CLI para desarrollo local
	@echo "Compilando para desarrollo local..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Binario generado: $(BUILD_DIR)/$(BINARY_NAME)"

# Instalación local
install: build ## Instalar el binario localmente
	@echo "Instalando $(BINARY_NAME)..."
	go install $(LDFLAGS) .
	@echo "$(BINARY_NAME) instalado en $$(go env GOPATH)/bin/"

# Pruebas
test: ## Ejecutar pruebas
	@echo "Ejecutando pruebas..."
	go test -v ./...

test-coverage: ## Ejecutar pruebas con cobertura
	@echo "Ejecutando pruebas con cobertura..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Reporte de cobertura generado: coverage.html"

# Limpieza
clean: ## Limpiar archivos generados
	@echo "Limpiando archivos generados..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Desarrollo
dev-setup: ## Configurar entorno de desarrollo
	@echo "Configurando entorno de desarrollo..."
	go mod download
	go mod tidy
	@echo "Para compilar con GUI instala las dependencias del sistema:"
	@echo "  Ubuntu/Debian: sudo apt-get install libgl1-mesa-dev xorg-dev"
	@echo "  macOS: ya incluidas en Xcode Command Line Tools"
	@echo "  Windows: usa un entorno con CGO habilitado"

# Verificar dependencias
deps-check: ## Verificar dependencias
	@echo "Verificando dependencias..."
	go mod verify
	go mod download

# Lint
lint: ## Ejecutar linter
	@echo "Ejecutando linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint no está instalado. Instálalo con:"; \
		echo "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Formateo
fmt: ## Formatear código
	@echo "Formateando código..."
	go fmt ./...
	go mod tidy

# Release
release: clean test build-cli ## Crear release completo
	@echo "Creando release v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@for binary in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$binary" ]; then \
			echo "Empaquetando $$binary..."; \
			cp "$$binary" $(BUILD_DIR)/release/; \
		fi \
	done
	@echo "Release v$(VERSION) creado en $(BUILD_DIR)/release/"

# Docker
docker-build: ## Construir imagen Docker
	@echo "Construyendo imagen Docker..."
	docker build -t osdo-cli:$(VERSION) .
	docker tag osdo-cli:$(VERSION) osdo-cli:latest

docker-run: ## Ejecutar en Docker
	@echo "Ejecutando en Docker..."
	docker run --rm -it osdo-cli:latest

# Información
info: ## Mostrar información del proyecto
	@echo "Proyecto: OSDO DevSecOps CLI"
	@echo "Versión: $(VERSION)"
	@echo "Go version: $$(go version)"
	@echo "Build dir: $(BUILD_DIR)"
	@echo ""
	@echo "Compilación CLI (sin GUI):"
	@echo "  make build-cli    - Binarios multiplataforma"
	@echo "  make build        - Binario local"
	@echo ""
	@echo "Compilación GUI (con Fyne):"
	@echo "  make build-gui    - Requiere CGO y dependencias"
	@echo ""
	@echo "Desarrollo:"
	@echo "  make dev-setup    - Configurar entorno"
	@echo "  make test         - Ejecutar pruebas"
	@echo "  make lint         - Ejecutar linter"

# Default target
.DEFAULT_GOAL := help
