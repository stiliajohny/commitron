# Docker Setup for Open WebUI

This Docker Compose setup provides a complete local AI environment with Ollama and Open WebUI that can be used with commitron's custom provider feature.

## Prerequisites

- Docker and Docker Compose installed
- At least 4GB of available RAM (8GB recommended)
- At least 10GB of available disk space

## Quick Start

### Option 1: Automated Setup (Recommended)

1. **Clone the repository and navigate to the project directory:**
   ```bash
   cd commitron
   ```

2. **Run the setup command:**
   ```bash
   make docker-setup
   ```

The command will automatically:
- Check prerequisites (Docker, Docker Compose)
- Create the environment file
- Start the services
- Provide next steps

### Option 2: Manual Setup

1. **Clone the repository and navigate to the project directory:**
   ```bash
   cd commitron
   ```

2. **Copy the environment file:**
   ```bash
   cp docker-compose.env.example .env
   ```

3. **Start the services:**
   ```bash
   docker-compose up -d
   ```

4. **Access Open WebUI:**
   - Open your browser and go to `http://localhost:3000`
   - The default port can be changed by modifying the `OPEN_WEBUI_PORT` variable in `.env`

5. **Pull a model in Ollama:**
   ```bash
   # Pull a model using make command
   make docker-pull-model MODEL=mistral:latest
   
   # Or pull other models
   make docker-pull-model MODEL=llama2:latest
   make docker-pull-model MODEL=codellama:latest
   
   # List available models
   make docker-list-models
   ```

## Configure commitron to use Open WebUI

Once Open WebUI is running, you can configure commitron to use it as a custom provider:

1. **Create or edit your commitron configuration:**
   ```bash
   commitron init
   ```

2. **Update the configuration file (`~/.commitronrc`):**
   ```yaml
   ai:
     provider: custom
     api_endpoint: http://localhost:3000/v1/chat/completions
     model: mistral:latest  # Use the model name as configured in Open WebUI
     api_key: ""  # Usually not required for local instances
     temperature: 0.7
     max_tokens: 1000
   ```

3. **Test the setup:**
   ```bash
   # Stage some files
   git add .
   
   # Generate a commit message using the custom provider
   commitron generate
   ```

## Available Make Commands

The project includes several Make commands to simplify Docker operations:

| Command | Description |
|---------|-------------|
| `make docker-setup` | Complete setup (env file + start services) |
| `make docker-up` | Start services only |
| `make docker-down` | Stop services |
| `make docker-pull-model MODEL=name` | Pull a model in Ollama |
| `make docker-list-models` | List available models |
| `make docker-status` | Show service status |
| `make docker-logs` | Show all logs |
| `make docker-logs-ollama` | Show Ollama logs only |
| `make docker-logs-webui` | Show Open WebUI logs only |
| `make docker-clean` | Stop and remove all data |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_DOCKER_TAG` | `latest` | Ollama Docker image tag |
| `WEBUI_DOCKER_TAG` | `main` | Open WebUI Docker image tag |
| `OPEN_WEBUI_PORT` | `3000` | Port for Open WebUI web interface |
| `WEBUI_SECRET_KEY` | (empty) | Secret key for Open WebUI (optional) |

## Available Models

You can use any model that's compatible with Ollama. Some popular models for code-related tasks:

- `mistral:latest` - Good general-purpose model
- `codellama:latest` - Specialized for code generation
- `llama2:latest` - Meta's Llama 2 model
- `neural-chat:latest` - Intel's optimized model

To see all available models:
```bash
docker exec -it ollama ollama list
```

## Troubleshooting

### Port already in use
If port 3000 is already in use, change the `OPEN_WEBUI_PORT` in your `.env` file:
```bash
OPEN_WEBUI_PORT=3001
```

### Out of memory
If you encounter memory issues:
1. Pull smaller models (e.g., `mistral:7b` instead of `mistral:latest`)
2. Increase Docker memory limits in Docker Desktop settings
3. Close other memory-intensive applications

### Services not starting
Check the logs:
```bash
make docker-logs
# Or check specific services:
make docker-logs-ollama
make docker-logs-webui
```

### API endpoint not accessible
Ensure Open WebUI is fully started:
```bash
make docker-status
```

The Open WebUI service should show as "Up" before trying to use it with commitron.

## Stopping the Services

To stop all services:
```bash
make docker-down
```

To stop and remove all data (volumes):
```bash
make docker-clean
```

## Data Persistence

The setup uses Docker volumes to persist:
- Ollama models and data (`ollama` volume)
- Open WebUI configuration and data (`open-webui` volume)

Your models and settings will persist between container restarts.
