#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Jan Server Mono Setup ===${NC}"

# Check prerequisites
echo -e "\n${YELLOW}Checking prerequisites...${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is installed${NC}"

# Check Docker Compose
if ! docker compose version &> /dev/null; then
    echo -e "${RED}Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker Compose is installed${NC}"

# Check Go (optional for development)
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✓ Go is installed ($GO_VERSION)${NC}"
else
    echo -e "${YELLOW}⚠ Go is not installed. Required only for local development.${NC}"
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "\n${YELLOW}Creating .env file from template...${NC}"
    if [ -f .env.template ]; then
        cp .env.template .env
        echo -e "${GREEN}✓ Created .env file${NC}"
        echo -e "${YELLOW}⚠ Please edit .env file with your configuration${NC}"
    else
        echo -e "${RED}✗ .env.template not found${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ .env file already exists${NC}"
fi

# Create necessary directories
echo -e "\n${YELLOW}Creating directories...${NC}"
mkdir -p data/postgres
mkdir -p data/minio
mkdir -p data/redis
mkdir -p logs
echo -e "${GREEN}✓ Directories created${NC}"

# Download Go dependencies
if command -v go &> /dev/null; then
    echo -e "\n${YELLOW}Downloading Go dependencies...${NC}"
    cd apps/backend
    go mod download
    cd ../..
    echo -e "${GREEN}✓ Go dependencies downloaded${NC}"
fi

echo -e "\n${GREEN}=== Setup Complete ===${NC}"
echo -e "\nNext steps:"
echo -e "  1. Edit .env file with your configuration"
echo -e "  2. Run 'make up' to start all services"
echo -e "  3. Run 'make health-check' to verify services are running"
