#!/bin/bash

# Deploy tactical dashboard script
# Uploads tactical_dashboard.html to the Cloudflare R2 bucket that serves
# gefunkt.dev, as the object "travel" (served at https://gefunkt.dev/travel).
# Authenticates with your Cloudflare login (run "npx wrangler login" once)
# or CLOUDFLARE_API_TOKEN; no R2 keys are needed.

set -e  # Exit on any error

# Pinned past the 30-day supply-chain soak window
WRANGLER="npx --yes wrangler@4.137.0"

OBJECT_KEY="travel"

# Function to load optional settings from .env file
load_env() {
    # Only R2_BUCKET and R2_ACCOUNT_ID are read; other secrets stay out of the
    # environment
    if [[ -f .env ]]; then
        echo "Loading R2_BUCKET and R2_ACCOUNT_ID from .env file..."
        R2_BUCKET=${R2_BUCKET:-$(grep -E '^R2_BUCKET=' .env | cut -d= -f2-)}
        R2_ACCOUNT_ID=${R2_ACCOUNT_ID:-$(grep -E '^R2_ACCOUNT_ID=' .env | cut -d= -f2-)}
    fi

    R2_BUCKET=${R2_BUCKET:-gefunkt-data}
    echo "Using bucket: $R2_BUCKET"

    # Lets wrangler skip the account picker when your login has several accounts
    if [[ -n "$R2_ACCOUNT_ID" ]]; then
        export CLOUDFLARE_ACCOUNT_ID="$R2_ACCOUNT_ID"
    fi
}

# Function to check if required files and tools exist
check_files() {
    if [[ ! -f "tactical_dashboard.html" ]]; then
        echo "Error: tactical_dashboard.html not found in current directory"
        exit 1
    fi

    if ! command -v npx >/dev/null 2>&1; then
        echo "Error: npx not found"
        echo "Please install Node.js to run wrangler"
        exit 1
    fi

    echo "Files found: tactical_dashboard.html"
}

# Function to deploy dashboard
deploy_dashboard() {
    echo "Deploying tactical_dashboard.html..."
    echo "Command: wrangler r2 object put ${R2_BUCKET}/${OBJECT_KEY} --file tactical_dashboard.html --remote"

    if $WRANGLER r2 object put "${R2_BUCKET}/${OBJECT_KEY}" \
        --file tactical_dashboard.html \
        --content-type "text/html; charset=utf-8" \
        --cache-control "no-cache" \
        --remote; then
        echo "✅ Dashboard deployed successfully!"
        echo "Deployed to: r2://${R2_BUCKET}/${OBJECT_KEY} (https://gefunkt.dev/travel)"
    else
        echo "❌ Deployment failed!"
        echo "If you are not logged in, run: npx wrangler@4.137.0 login"
        exit 1
    fi
}

# Main execution
main() {
    echo "🚀 Starting tactical dashboard deployment..."
    echo

    load_env
    check_files
    echo
    deploy_dashboard
    echo
    echo "🎉 Deployment complete!"
}

# Run main function
main "$@"
