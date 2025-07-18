#!/bin/bash

# Kind Cluster Deployment Script
# This script creates a kind cluster and can be run multiple times safely

set -euo pipefail

# Configuration
CLUSTER_NAME=${KIND_CLUSTER_NAME:-"crib"}
CONFIG_FILE=${KIND_CONFIG_FILE:-"kind.defaults.yaml"}

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

    # Check if kind is installed
    if ! command -v kind &> /dev/null; then
        log_error "kind is not installed. Please install kind first."
        echo "Visit: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        echo "Visit: https://kubernetes.io/docs/tasks/tools/install-kubectl/"
        exit 1
    fi

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    log_success "All prerequisites are satisfied"
}

# Check if cluster exists
cluster_exists() {
    kind get clusters | grep -q "^${CLUSTER_NAME}$"
}

# Create cluster
create_cluster() {
    log_info "Creating kind cluster '${CLUSTER_NAME}'..."

    if [ -f "$CONFIG_FILE" ]; then
        log_info "Using configuration file: $CONFIG_FILE"
        kind create cluster --name "$CLUSTER_NAME" --config "$CONFIG_FILE"
    else
        log_info "Using default configuration"
        kind create cluster --name "$CLUSTER_NAME"
    fi

    log_success "Cluster '${CLUSTER_NAME}' created successfully!"
}

# Configure kubectl context
configure_kubectl() {
    log_info "Configuring kubectl context..."
    kind export kubeconfig --name "$CLUSTER_NAME"
    log_success "Kubectl context set to cluster '${CLUSTER_NAME}'"
}

# Wait for cluster to be ready
wait_for_cluster() {
    log_info "Waiting for cluster to be ready..."

    # Wait for nodes to be ready
    kubectl wait --for=condition=Ready nodes --all --timeout=300s

    # Wait for core components
    kubectl wait --for=condition=Available deployment/coredns -n kube-system --timeout=300s

    log_success "Cluster is ready!"
}

# Show cluster status
show_status() {
    log_info "Cluster status:"
    echo "Cluster name: $CLUSTER_NAME"
    echo "Kubernetes version: $(kubectl version --client)"
    echo ""
    kubectl cluster-info
    echo ""
    kubectl get nodes
}

# Main deployment function
deploy_cluster() {
    log_info "Starting kind cluster deployment..."

    # Check if cluster already exists
    if cluster_exists; then
        log_warning "Cluster '${CLUSTER_NAME}' already exists. Skipping creation."
        log_info "To recreate the cluster, run: kind delete cluster --name ${CLUSTER_NAME}"
        configure_kubectl
        show_status
        exit 0
    fi

    # Create cluster
    create_cluster

    # Configure kubectl
    configure_kubectl

    # Wait for cluster to be ready
    wait_for_cluster

    # Show final status
    show_status

    log_success "Kind cluster deployment completed successfully!"
}

# Delete cluster function
delete_cluster() {
    log_info "Deleting kind cluster..."

    if cluster_exists; then
        kind delete cluster --name "$CLUSTER_NAME"
        log_success "Cluster '${CLUSTER_NAME}' deleted successfully!"
    else
        log_warning "Cluster '${CLUSTER_NAME}' does not exist."
    fi
}

# Show cluster status function
show_cluster_status() {
    log_info "Available kind clusters:"
    kind get clusters

    echo ""
    if cluster_exists; then
        log_info "Cluster '${CLUSTER_NAME}' details:"
        configure_kubectl
        show_status
    else
        log_warning "Cluster '${CLUSTER_NAME}' does not exist."
    fi
}

# Help function
show_help() {
    cat << EOF
Idempotent Kind Cluster Deployment Script

Usage: $0 [COMMAND]

Commands:
    deploy     Deploy the kind cluster (default)
    delete     Delete the kind cluster
    status     Show cluster status
    help       Show this help message

Environment Variables:
    KIND_CLUSTER_NAME     Cluster name (default: cop-tools)
    KIND_CLUSTER_CONFIG   Path to kind config file (default: kind-config.yaml)
    KIND_K8S_VERSION      Kubernetes version (default: v1.28.0)

Examples:
    $0                   # Deploy cluster with default settings
    $0 deploy            # Deploy cluster
    $0 delete            # Delete cluster
    $0 status            # Show cluster status
    KIND_CLUSTER_NAME=my-cluster $0  # Deploy with custom cluster name

EOF
}

# Main script logic
main() {
    local command=${1:-deploy}

    case "$command" in
        deploy)
            check_prerequisites
            deploy_cluster
            ;;
        delete)
            delete_cluster
            ;;
        status)
            show_cluster_status
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
