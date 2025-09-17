#!/bin/bash

# Deploy tactical dashboard script
# Deploys tactical_dashboard.html using SCP with configuration from .env file

set -e  # Exit on any error

# Function to load environment variables from .env file
load_env() {
    if [[ -f .env ]]; then
        echo "Loading environment variables from .env file..."
        # Export variables from .env file, ignoring comments and empty lines
        export $(grep -v '^#' .env | grep -v '^$' | xargs)
    else
        echo "Error: .env file not found in current directory"
        exit 1
    fi
}

# Function to validate required variables
validate_env() {
    if [[ -z "$DEPLOY_URL" ]]; then
        echo "Error: DEPLOY_URL not found in .env file"
        echo "Please ensure DEPLOY_URL is set in .env file (format: user@host:path)"
        exit 1
    fi

    echo "Using DEPLOY_URL: $DEPLOY_URL"
}

# Function to check if required files exist
check_files() {
    if [[ ! -f "tactical_dashboard.html" ]]; then
        echo "Error: tactical_dashboard.html not found in current directory"
        exit 1
    fi

    if [[ ! -f "deploy.pem" ]]; then
        echo "Error: deploy.pem not found in current directory"
        echo "Please ensure the SSH private key file is present"
        exit 1
    fi

    echo "Files found: tactical_dashboard.html and deploy.pem"
}

# Function to deploy dashboard
deploy_dashboard() {
    echo "Deploying tactical_dashboard.html..."
    echo "Command: scp -i deploy.pem tactical_dashboard.html ${DEPLOY_URL}/tactical_dashboard.html"

    if scp -i deploy.pem tactical_dashboard.html "${DEPLOY_URL}/tactical_dashboard.html"; then
        echo "✅ Dashboard deployed successfully!"
        echo "Deployed to: ${DEPLOY_URL}/tactical_dashboard.html"
    else
        echo "❌ Deployment failed!"
        exit 1
    fi
}

# Main execution
main() {
    echo "🚀 Starting tactical dashboard deployment..."
    echo

    load_env
    validate_env
    check_files
    echo
    deploy_dashboard
    echo
    echo "🎉 Deployment complete!"
}

# Run main function
main "$@"