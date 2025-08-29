# Variables
BINARY_NAME=commitron
BUILD_DIR=bin
DIST_DIR=dist
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 windows/amd64

# Default target
.DEFAULT_GOAL := help

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

# Check if Go is installed
check-go: ## Check if Go is installed
	@if ! command -v go &> /dev/null; then \
		echo "Go is not installed or not in PATH"; \
		exit 1; \
	fi
	@go version

# Get dependencies
deps: ## Get Go dependencies
	go mod tidy

# Run tests
test: ## Run Go tests
	go test -v ./...

# Build for current platform
build: check-go deps ## Build for current platform
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Build successful!"

# Build for all platforms
build-all: check-go deps ## Build for all supported platforms
	@echo "Building binaries for all platforms..."
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output_name="$(BINARY_NAME)"; \
		if [ "$$GOOS" = "windows" ]; then \
			output_name="$(BINARY_NAME).exe"; \
		fi; \
		output_path="$(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH"; \
		if [ "$$GOOS" = "windows" ]; then \
			output_path="$$output_path.exe"; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -o "$$output_path" ./cmd/$(BINARY_NAME); \
	done
	@echo "Build completed successfully!"
	@ls -la $(DIST_DIR)

# Run the binary
run: build ## Run the binary with provided arguments
	@echo "Running $(BINARY_NAME)..."
	@echo "-------------------------------------"
	@./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Clean build artifacts
clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

# Docker commands
docker-check: ## Check if Docker and Docker Compose are installed
	@if ! command -v docker &> /dev/null; then \
		echo "❌ Docker is not installed. Please install Docker first."; \
		exit 1; \
	fi
	@if ! command -v docker-compose &> /dev/null; then \
		echo "❌ Docker Compose is not installed. Please install Docker Compose first."; \
		exit 1; \
	fi
	@echo "✅ Docker and Docker Compose are installed"

docker-setup: docker-check ## Setup Docker environment (copy env file and start services)
	@echo "🚀 Setting up Docker environment for commitron..."
	@if [ ! -f ".env" ]; then \
		if [ -f "docker-compose.env.example" ]; then \
			echo "📋 Copying environment file..."; \
			cp docker-compose.env.example .env; \
			echo "✅ Environment file created. You can edit .env to customize settings."; \
		else \
			echo "⚠️  docker-compose.env.example not found. Creating basic .env file..."; \
			echo "# Docker Compose environment variables" > .env; \
			echo "OLLAMA_DOCKER_TAG=latest" >> .env; \
			echo "WEBUI_DOCKER_TAG=main" >> .env; \
			echo "OPEN_WEBUI_PORT=3000" >> .env; \
			echo "WEBUI_SECRET_KEY=" >> .env; \
		fi; \
	fi
	@echo "🐳 Starting Docker services..."
	@docker-compose up -d
	@echo "⏳ Waiting for services to start..."
	@sleep 10
	@if ! docker-compose ps | grep -q "Up"; then \
		echo "❌ Services failed to start. Check logs with: make docker-logs"; \
		exit 1; \
	fi
	@echo "✅ Services started successfully!"
	@echo ""
	@echo "🎉 Setup complete! Next steps:"
	@echo ""
	@echo "1. Pull a model in Ollama:"
	@echo "   make docker-pull-model MODEL=mistral:latest"
	@echo ""
	@echo "2. Access Open WebUI:"
	@echo "   http://localhost:3000"
	@echo ""
	@echo "3. Configure commitron to use the custom provider:"
	@echo "   commitron init"
	@echo ""
	@echo "4. Edit ~/.commitronrc and set:"
	@echo "   provider: custom"
	@echo "   api_endpoint: http://localhost:3000/v1/chat/completions"
	@echo ""
	@echo "5. Test the setup:"
	@echo "   git add ."
	@echo "   commitron generate"
	@echo ""
	@echo "📖 For detailed instructions, see DOCKER_SETUP.md"
	@echo ""
	@echo "🛑 To stop services: make docker-down"

docker-up: docker-check ## Start Docker services
	@echo "🐳 Starting Docker services..."
	@docker-compose up -d

docker-down: ## Stop Docker services
	@echo "🛑 Stopping Docker services..."
	@docker-compose down

docker-logs: ## Show Docker service logs
	@docker-compose logs

docker-logs-ollama: ## Show Ollama service logs
	@docker-compose logs ollama

docker-logs-webui: ## Show Open WebUI service logs
	@docker-compose logs open-webui

docker-pull-model: docker-check ## Pull a model in Ollama (usage: make docker-pull-model MODEL=mistral:latest)
	@if [ -z "$(MODEL)" ]; then \
		echo "❌ Please specify a model: make docker-pull-model MODEL=mistral:latest"; \
		exit 1; \
	fi
	@echo "📥 Pulling model $(MODEL) in Ollama..."
	@docker exec -it ollama ollama pull $(MODEL)

docker-list-models: docker-check ## List available models in Ollama
	@echo "📋 Available models in Ollama:"
	@docker exec -it ollama ollama list

docker-status: ## Show Docker services status
	@echo "📊 Docker services status:"
	@docker-compose ps

docker-clean: ## Stop services and remove volumes (WARNING: This will delete all data)
	@echo "⚠️  WARNING: This will stop services and remove all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "🧹 Cleaning up Docker environment..."; \
		docker-compose down -v; \
		echo "✅ Cleanup complete"; \
	else \
		echo "❌ Cleanup cancelled"; \
	fi

.PHONY: help check-go deps test build build-all run clean docker-check docker-setup docker-up docker-down docker-logs docker-logs-ollama docker-logs-webui docker-pull-model docker-list-models docker-status docker-clean