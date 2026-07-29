#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  publish-extension.sh [--pat PAT] [--version X.Y.Z]

Environment:
  AZURE_DEVOPS_MARKETPLACE_PAT  Marketplace PAT used when --pat is omitted.
  NPM_CONFIG_CACHE              Optional npm cache path.

Examples:
  AZURE_DEVOPS_MARKETPLACE_PAT=... publish-extension.sh
  publish-extension.sh --pat ... --version 1.0.2
USAGE
}

PAT="${AZURE_DEVOPS_MARKETPLACE_PAT:-}"
REQUESTED_VERSION=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pat)
      PAT="${2:-}"
      shift 2
      ;;
    --version)
      REQUESTED_VERSION="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$PAT" ]]; then
  echo "Marketplace PAT is required. Pass --pat or set AZURE_DEVOPS_MARKETPLACE_PAT." >&2
  exit 2
fi

for cmd in node npm jq tfx; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Required command not found: $cmd" >&2
    exit 2
  fi
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
TASK_DIR="$ROOT_DIR/Smurf"

cd "$ROOT_DIR"

CURRENT_VERSION="$(jq -r '.version' vss-extension.json)"

if [[ -n "$REQUESTED_VERSION" ]]; then
  NEXT_VERSION="$REQUESTED_VERSION"
else
  IFS=. read -r MAJOR MINOR PATCH <<<"$CURRENT_VERSION"
  NEXT_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
fi

if [[ ! "$NEXT_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Version must use X.Y.Z format: $NEXT_VERSION" >&2
  exit 2
fi

node - "$NEXT_VERSION" <<'NODE'
const fs = require("fs");
const version = process.argv[2];
const [major, minor, patch] = version.split(".").map(Number);

function updateJson(file, mutate) {
  const data = JSON.parse(fs.readFileSync(file, "utf8"));
  mutate(data);
  fs.writeFileSync(file, `${JSON.stringify(data, null, 2)}\n`);
}

updateJson("vss-extension.json", (data) => {
  data.version = version;
  data.public = true;
});

updateJson("Smurf/task.json", (data) => {
  data.version = { Major: major, Minor: minor, Patch: patch };
});

updateJson("Smurf/package.json", (data) => {
  data.version = version;
});
NODE

export NPM_CONFIG_CACHE="${NPM_CONFIG_CACHE:-$ROOT_DIR/.npm-cache}"

echo "Preparing Smurf Azure DevOps extension $NEXT_VERSION"
npm --prefix "$TASK_DIR" install --package-lock-only
npm --prefix "$TASK_DIR" ci

jq empty vss-extension.json "$TASK_DIR/task.json" "$TASK_DIR/package.json" "$TASK_DIR/package-lock.json"
node --check "$TASK_DIR/index.js"
npm --prefix "$TASK_DIR" audit

tfx extension create --manifest-globs vss-extension.json

tfx extension publish \
  --manifest-globs vss-extension.json \
  --no-wait-validation \
  --token "$PAT"

echo "Published clouddrove.smurf-azure-pipelines $NEXT_VERSION"
echo "Check validation:"
echo "tfx extension isvalid --publisher clouddrove --extension-id smurf-azure-pipelines --version $NEXT_VERSION --service-url https://marketplace.visualstudio.com/ --token '<PAT>'"
