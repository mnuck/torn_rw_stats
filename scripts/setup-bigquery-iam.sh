#!/usr/bin/env bash
# setup-bigquery-iam.sh — Grant the app's service account the IAM roles needed
# to stream-insert data into BigQuery.
#
# Prerequisites:
#   - gcloud CLI installed and authenticated as a project owner or editor
#   - jq installed (for parsing the credentials JSON)
#
# Usage:
#   export BIGQUERY_PROJECT_ID=my-gcp-project
#   ./scripts/setup-bigquery-iam.sh
#
# The script reads GOOGLE_CREDENTIALS_FILE (defaults to credentials.json) to
# find the service account email, then grants:
#   roles/bigquery.user        — project-level: run jobs, list datasets
#   roles/bigquery.dataEditor  — project-level: create tables, stream inserts

set -euo pipefail

CREDENTIALS_FILE="${GOOGLE_CREDENTIALS_FILE:-credentials.json}"
PROJECT_ID="${BIGQUERY_PROJECT_ID:-}"

# --- Validate prerequisites ---

if ! command -v gcloud &>/dev/null; then
  echo "ERROR: gcloud CLI not found. Install it from https://cloud.google.com/sdk/docs/install" >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq not found. Install it with your package manager (e.g. brew install jq)" >&2
  exit 1
fi

if [[ ! -f "${CREDENTIALS_FILE}" ]]; then
  echo "ERROR: Credentials file '${CREDENTIALS_FILE}' not found." >&2
  echo "       Set GOOGLE_CREDENTIALS_FILE to the correct path." >&2
  exit 1
fi

# --- Extract service account email ---

SA_EMAIL=$(jq -r '.client_email' "${CREDENTIALS_FILE}")
if [[ -z "${SA_EMAIL}" || "${SA_EMAIL}" == "null" ]]; then
  echo "ERROR: Could not extract client_email from '${CREDENTIALS_FILE}'." >&2
  echo "       Make sure it is a valid service account key file." >&2
  exit 1
fi

echo "Service account: ${SA_EMAIL}"

# --- Resolve project ID ---

if [[ -z "${PROJECT_ID}" ]]; then
  # Fall back to the project embedded in the credentials file
  PROJECT_ID=$(jq -r '.project_id' "${CREDENTIALS_FILE}")
fi

if [[ -z "${PROJECT_ID}" || "${PROJECT_ID}" == "null" ]]; then
  echo "ERROR: BIGQUERY_PROJECT_ID is not set and could not be read from the credentials file." >&2
  exit 1
fi

echo "Project:         ${PROJECT_ID}"
echo ""

# --- Grant IAM roles ---

grant_role() {
  local ROLE="$1"
  echo "Granting ${ROLE}..."
  gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="${ROLE}" \
    --quiet
  echo "  OK"
}

grant_role "roles/bigquery.user"
grant_role "roles/bigquery.dataEditor"

echo ""
echo "Done. The service account '${SA_EMAIL}' now has BigQuery access on project '${PROJECT_ID}'."
echo ""
echo "Next step: create the dataset and table by running:"
echo "  go run ./cmd/setup-bigquery/"
