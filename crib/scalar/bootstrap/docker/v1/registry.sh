#!/bin/bash

# Docker Registry Deployment Script
# This script creates a local docker registry container and can be run multiple times safely

set -euo pipefail

# Configuration
REGISTRY_NAME=${REGISTRY_NAME:-"registry"}
REGISTRY_PORT=${REGISTRY_PORT:-"5001"}
REGISTRY_IMAGE=${REGISTRY_IMAGE:-"registry:2"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    log_success "All prerequisites are satisfied"
}

# Check if registry container exists and is running
registry_exists() {
    docker ps -a --format "table {{.Names}}" | grep -q "^${REGISTRY_NAME}$"
}

registry_running() {
    docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null | grep -q "true"
}

# Create registry container
create_registry() {
    log_info "Creating docker registry container '${REGISTRY_NAME}'..."

    docker run -d \
        --restart=always \
        -p "127.0.0.1:${REGISTRY_PORT}:5000" \
        --network bridge \
        --name "${REGISTRY_NAME}" \
        "${REGISTRY_IMAGE}"

    log_success "Registry container '${REGISTRY_NAME}' created successfully!"
}

# Start registry container if it exists but is not running
start_registry() {
    log_info "Starting existing registry container '${REGISTRY_NAME}'..."
    docker start "${REGISTRY_NAME}"
    log_success "Registry container '${REGISTRY_NAME}' started successfully!"
}

# Wait for registry to be ready
wait_for_registry() {
    log_info "Waiting for registry to be ready..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s "http://127.0.0.1:${REGISTRY_PORT}/v2/" > /dev/null 2>&1; then
            log_success "Registry is ready!"
            return 0
        fi

        log_info "Attempt $attempt/$max_attempts: Registry not ready yet, waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "Registry failed to become ready after $max_attempts attempts"
    return 1
}

# Show registry status
show_status() {
    log_info "Registry status:"
    echo "Container name: $REGISTRY_NAME"
    echo "Port: $REGISTRY_PORT"
    echo "Image: $REGISTRY_IMAGE"
    echo ""

    if registry_exists; then
        echo "Container details:"
        docker ps -a --filter "name=${REGISTRY_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        echo ""

        if registry_running; then
            echo "Registry endpoint: http://127.0.0.1:${REGISTRY_PORT}"
            echo "Registry API: http://127.0.0.1:${REGISTRY_PORT}/v2/"
        fi
    else
        log_warning "Registry container does not exist"
    fi
}

# Main deployment function
deploy_registry() {
    log_info "Starting docker registry deployment..."

    # Check if registry already exists and is running
    if registry_exists; then
        if registry_running; then
            log_warning "Registry container '${REGISTRY_NAME}' already exists and is running. Skipping creation."
            show_status
            exit 0
        else
            log_warning "Registry container '${REGISTRY_NAME}' exists but is not running. Starting it..."
            start_registry
        fi
    else
        # Create registry
        create_registry
    fi

    # Wait for registry to be ready
    wait_for_registry

    # Show final status
    show_status

    log_success "Docker registry deployment completed successfully!"
}

# Delete registry function
delete_registry() {
    log_info "Deleting docker registry container..."

    if registry_exists; then
        docker stop "${REGISTRY_NAME}" 2>/dev/null || true
        docker rm "${REGISTRY_NAME}"
        log_success "Registry container '${REGISTRY_NAME}' deleted successfully!"
    else
        log_warning "Registry container '${REGISTRY_NAME}' does not exist."
    fi
}

# Show registry status function
show_registry_status() {
    log_info "Available registry containers:"
    docker ps -a --filter "ancestor=${REGISTRY_IMAGE}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

    echo ""
    if registry_exists; then
        log_info "Registry '${REGISTRY_NAME}' details:"
        show_status
    else
        log_warning "Registry '${REGISTRY_NAME}' does not exist."
    fi
}

# Help function
show_help() {
    cat << EOF
Idempotent Docker Registry Deployment Script

Usage: $0 [COMMAND]

Commands:
    deploy     Deploy the docker registry (default)
    delete     Delete the docker registry
    status     Show registry status
    help       Show this help message

Environment Variables:
    REGISTRY_NAME     Registry container name (default: registry)
    REGISTRY_PORT     Registry port (default: 5001)
    REGISTRY_IMAGE    Registry image (default: registry:2)

Examples:
    $0                   # Deploy registry with default settings
    $0 deploy            # Deploy registry
    $0 delete            # Delete registry
    $0 status            # Show registry status
    REGISTRY_PORT=5002 $0  # Deploy with custom port

EOF
}

# Main script logic
main() {
    local command=${1:-deploy}

    case "$command" in
        deploy)
            check_prerequisites
            deploy_registry
            ;;
        delete)
            delete_registry
            ;;
        status)
            show_registry_status
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
