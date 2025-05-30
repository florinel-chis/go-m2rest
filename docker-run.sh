#!/bin/bash

# Docker helper script for go-m2rest

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
DEFAULT_HOST="http://magento.local"
DEFAULT_STORE_CODE="all"

# Function to show usage
show_usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  test              Run all tests"
    echo "  test-specific     Run specific test (set TEST_NAME env var)"
    echo "  test-quick        Run quick connectivity test"
    echo "  bulk-create       Create bulk products"
    echo "  bulk-update       Update stock from CSV"
    echo "  shell             Open shell in container"
    echo "  build             Build Docker images"
    echo ""
    echo "Environment variables:"
    echo "  MAGENTO_HOST          Magento URL (default: $DEFAULT_HOST)"
    echo "  MAGENTO_BEARER_TOKEN  Integration token (required)"
    echo "  MAGENTO_STORE_CODE    Store code (default: $DEFAULT_STORE_CODE)"
    echo "  TEST_NAME             Specific test to run"
    echo ""
    echo "Examples:"
    echo "  # Run all tests"
    echo "  MAGENTO_BEARER_TOKEN=abc123 ./docker-run.sh test"
    echo ""
    echo "  # Run specific test"
    echo "  MAGENTO_BEARER_TOKEN=abc123 TEST_NAME=TestAdvancedProducts_VirtualProduct ./docker-run.sh test-specific"
    echo ""
    echo "  # Create 50 products"
    echo "  MAGENTO_BEARER_TOKEN=abc123 ./docker-run.sh bulk-create 50"
    echo ""
}

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    exit 1
fi

# Check for bearer token
if [ -z "$MAGENTO_BEARER_TOKEN" ] && [ "$1" != "build" ] && [ "$1" != "help" ]; then
    echo -e "${YELLOW}Warning: MAGENTO_BEARER_TOKEN not set${NC}"
    echo "You can set it with: export MAGENTO_BEARER_TOKEN=your_token_here"
    echo ""
fi

# Set default values if not provided
export MAGENTO_HOST=${MAGENTO_HOST:-$DEFAULT_HOST}
export MAGENTO_STORE_CODE=${MAGENTO_STORE_CODE:-$DEFAULT_STORE_CODE}

# Main command handling
case "$1" in
    test)
        echo -e "${GREEN}Running all tests...${NC}"
        docker-compose run --rm test
        ;;
        
    test-docker)
        echo -e "${GREEN}Running tests with direct Docker (with host mapping)...${NC}"
        docker build -t go-m2rest . && docker run --rm \            
            --add-host=magento.local:host-gateway \
            -e MAGENTO_HOST="${MAGENTO_HOST}" \
            -e MAGENTO_BEARER_TOKEN="${MAGENTO_BEARER_TOKEN}" \
            -e MAGENTO_STORE_CODE="${MAGENTO_STORE_CODE}" \
            go-m2rest
        ;;
        
    test-specific)
        if [ -z "$TEST_NAME" ]; then
            echo -e "${YELLOW}TEST_NAME not set, running API connection test${NC}"
            export TEST_NAME="TestFunctionalV2_APIConnection"
        fi
        echo -e "${GREEN}Running test: $TEST_NAME${NC}"
        docker-compose run --rm test-specific
        ;;
        
    test-quick)
        echo -e "${GREEN}Running quick connectivity test...${NC}"
        export TEST_NAME="TestFunctionalV2_APIConnection"
        docker-compose run --rm test-specific
        ;;
        
    bulk-create)
        COUNT=${2:-100}
        echo -e "${GREEN}Creating $COUNT products...${NC}"
        docker-compose run --rm bulk-update ./run_bulk_update.sh stock_updates.csv $COUNT 5 create
        ;;
        
    bulk-update)
        CSV=${2:-stock_updates.csv}
        echo -e "${GREEN}Updating stock from $CSV...${NC}"
        docker-compose run --rm bulk-update ./run_bulk_update.sh $CSV 0 5 update
        ;;
        
    shell)
        echo -e "${GREEN}Opening shell in container...${NC}"
        docker-compose run --rm dev /bin/sh
        ;;
        
    build)
        echo -e "${GREEN}Building Docker images...${NC}"
        docker-compose build
        ;;
        
    help|--help|-h|"")
        show_usage
        ;;
        
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        show_usage
        exit 1
        ;;
esac