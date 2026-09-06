#!/usr/bin/env bash
#
# Exercises every smurf subcommand. Runnable from a terminal and from CI, so
# the same checks cover both paths people use.
#
#   ./test/e2e/all-commands.sh [path-to-smurf]
#
# Two tiers:
#
#   1. Discovery. The command tree is read out of the binary itself rather than
#      listed here, so a subcommand added tomorrow is covered without editing
#      this file. Every command must answer --help and exit 0, which catches
#      broken registration, a malformed flag default, and panics in init.
#
#   2. Execution. Commands that need no cloud credentials are actually run
#      against local fixtures. Anything needing a registry, cluster or cloud
#      account is out of scope here and stays in tier 1.
set -euo pipefail

SMURF="${1:-smurf}"
if [ -x "$SMURF" ]; then
  # Resolved to an absolute path because the tier 2 checks cd into temp
  # directories, where a relative ./smurf no longer exists.
  SMURF="$(cd "$(dirname "$SMURF")" && pwd)/$(basename "$SMURF")"
elif ! SMURF="$(command -v "$SMURF" 2>/dev/null)"; then
  SMURF="$(cd "$(dirname "$0")/../.." && pwd)/smurf"
fi
[ -x "$SMURF" ] || { echo "smurf binary not found or not executable: $SMURF"; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0 fail=0
failed_list=()

ok()   { pass=$((pass+1)); printf '  ok   %s\n' "$1"; }
bad()  { fail=$((fail+1)); failed_list+=("$1"); printf '  FAIL %s\n' "$1"; }

run() { # run <description> <cmd...>
  local desc="$1"; shift
  if out=$("$@" 2>&1); then ok "$desc"; else
    bad "$desc"
    printf '       command: %s\n' "$*"
    printf '       output : %s\n' "$(printf '%s' "$out" | head -5 | tr '\n' ' ')"
  fi
}

echo "==> smurf under test: $SMURF"
"$SMURF" version || true

# ---------------------------------------------------------------- tier 1 ----
# Walk the command tree by parsing "Available Commands" out of each --help.
discover() {
  local -a path=("$@")
  local line grab=0 name
  while IFS= read -r line; do
    case "$line" in
      *"Available Commands:"*) grab=1; continue ;;
    esac
    if [ "$grab" -eq 1 ]; then
      [ -z "${line// /}" ] && break
      name="$(printf '%s' "$line" | awk '{print $1}')"
      case "$name" in
        help|completion|Flags:|"") continue ;;
      esac
      printf '%s\n' "${path[*]:+${path[*]} }$name"
      discover "${path[@]}" "$name"
    fi
  done < <("$SMURF" "${path[@]}" --help 2>/dev/null)
}

echo
echo "==> tier 1: every subcommand answers --help"
mapfile -t COMMANDS < <(discover)
echo "    discovered ${#COMMANDS[@]} subcommands"
[ "${#COMMANDS[@]}" -ge 50 ] || { echo "discovery found only ${#COMMANDS[@]} commands; the tree parse is probably broken"; exit 1; }

for cmd in "${COMMANDS[@]}"; do
  # shellcheck disable=SC2086
  run "$cmd --help" "$SMURF" $cmd --help
done

# ---------------------------------------------------------------- tier 2 ----
echo
echo "==> tier 2: commands that run without cloud credentials"

run "version" "$SMURF" version

mkdir -p "$WORK/initdir" && pushd "$WORK/initdir" >/dev/null
run "init writes smurf.yaml" "$SMURF" init
if [ -f smurf.yaml ]; then ok "smurf.yaml exists"; else bad "smurf.yaml missing after init"; fi
popd >/dev/null

# terraform: null provider only, no backend, no cloud
if command -v terraform >/dev/null 2>&1; then
  mkdir -p "$WORK/tf" && cat > "$WORK/tf/main.tf" <<'TF'
terraform {
  required_version = ">= 0.14"
}
resource "null_resource" "example" {}
TF
  pushd "$WORK/tf" >/dev/null
  run "stf init"     "$SMURF" stf init
  run "stf validate" "$SMURF" stf validate
  run "stf format"   "$SMURF" stf format
  run "stf plan"     "$SMURF" stf plan
  popd >/dev/null
else
  echo "  skip terraform execution: terraform not installed"
fi

# helm: chart authoring and rendering need no cluster
mkdir -p "$WORK/helm" && pushd "$WORK/helm" >/dev/null
run "selm create"   "$SMURF" selm create demo
if [ -d demo ]; then
  run "selm lint"     "$SMURF" selm lint demo
  run "selm template" "$SMURF" selm template demo demo
else
  bad "selm create did not produce a chart directory"
fi
popd >/dev/null

# docker: build/tag/remove against the local daemon
if docker info >/dev/null 2>&1; then
  mkdir -p "$WORK/docker" && printf 'FROM alpine:3.24\nCMD ["true"]\n' > "$WORK/docker/Dockerfile"
  pushd "$WORK/docker" >/dev/null
  run "sdkr build"  "$SMURF" sdkr build smurf-e2e latest --file Dockerfile --context .
  run "sdkr tag"    "$SMURF" sdkr tag smurf-e2e:latest smurf-e2e:tagged
  run "sdkr remove" "$SMURF" sdkr remove smurf-e2e:tagged
  run "sdkr remove" "$SMURF" sdkr remove smurf-e2e:latest
  popd >/dev/null
else
  echo "  skip docker execution: no reachable daemon"
fi

# --------------------------------------------------------------- summary ----
echo
echo "==================================="
echo " passed: $pass"
echo " failed: $fail"
if [ "$fail" -gt 0 ]; then
  echo
  echo " failures:"
  printf '   - %s\n' "${failed_list[@]}"
  exit 1
fi
echo " all subcommands exercised"
