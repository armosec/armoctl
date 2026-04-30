#!/usr/bin/env bash
# Regenerates JSON schemas under internal/schema/data/ from
# cadashboardbe's docs/swagger.json and from armotypes Go struct tags.
#
# Inputs:
#   $SWAGGER_PATH - path to swagger.json (default: ../cadashboardbe/docs/swagger.json)
# Outputs:
#   internal/schema/data/*.json (one per resource)
#
# This is the v1 generator: it copies a hand-curated allowlist of definitions
# from swagger.json. Per-cluster plans add resources to RESOURCES below.

set -euo pipefail

SWAGGER_PATH="${SWAGGER_PATH:-../cadashboardbe/docs/swagger.json}"
OUT_DIR="$(dirname "$0")/../internal/schema/data"

# resource:swaggerDefinitionName
RESOURCES=(
  "incidents:RuntimeIncident"
)

if [[ ! -f "$SWAGGER_PATH" ]]; then
  echo "swagger not found at $SWAGGER_PATH" >&2
  exit 2
fi

for entry in "${RESOURCES[@]}"; do
  name="${entry%%:*}"
  defn="${entry##*:}"
  out="$OUT_DIR/$name.json"
  extracted=$(jq --arg d "$defn" '.definitions[$d] // (.components.schemas[$d] // null)' "$SWAGGER_PATH")
  if [[ "$extracted" == "null" ]]; then
    echo "definition $defn not found in swagger" >&2
    exit 3
  fi
  echo "$extracted" \
    | jq '. + {"$schema":"https://json-schema.org/draft/2020-12/schema"}' \
    > "$out.tmp"
  mv "$out.tmp" "$out"
  echo "wrote $out"
done
