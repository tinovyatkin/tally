#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <vscode|openvsx> <vsix> [<vsix> ...]" >&2
}

registry="${1:-}"
if [[ -z "$registry" ]]; then
  usage
  exit 2
fi
shift

if (( $# == 0 )); then
  usage
  exit 2
fi

max_attempts="${PUBLISH_MAX_ATTEMPTS:-6}"
initial_delay="${PUBLISH_INITIAL_DELAY_SECONDS:-15}"
between_files="${PUBLISH_BETWEEN_FILES_SECONDS:-20}"
bunx_bin="${BUNX_BIN:-bunx}"

for value in "$max_attempts" "$initial_delay" "$between_files"; do
  if [[ ! "$value" =~ ^[0-9]+$ ]]; then
    echo "publish retry settings must be non-negative integers" >&2
    exit 2
  fi
done
if (( max_attempts == 0 )); then
  echo "PUBLISH_MAX_ATTEMPTS must be greater than zero" >&2
  exit 2
fi

case "$registry" in
  vscode)
    : "${VSCE_PAT:?VSCE_PAT is required}"
    ;;
  openvsx)
    : "${OVSX_TOKEN:?OVSX_TOKEN is required}"
    ;;
  *)
    usage
    exit 2
    ;;
esac

already_published_pattern='already published|already exists'
transient_pattern='RequestBlockedException|resource .Concurrency.|exceeding usage|rate.?limit|too many requests|HTTP (408|409|425|429|5[0-9][0-9])|ECONNRESET|ETIMEDOUT|EAI_AGAIN|socket hang up|temporary unavailable|temporarily unavailable|service unavailable|gateway timeout'

publish_one() {
  local vsix="$1"
  case "$registry" in
    vscode)
      "$bunx_bin" @vscode/vsce publish --pat "$VSCE_PAT" --packagePath "$vsix"
      ;;
    openvsx)
      "$bunx_bin" ovsx publish --pat "$OVSX_TOKEN" "$vsix"
      ;;
  esac
}

vsixes=("$@")
for index in "${!vsixes[@]}"; do
  vsix="${vsixes[$index]}"
  if [[ ! -f "$vsix" ]]; then
    echo "VSIX does not exist: $vsix" >&2
    exit 2
  fi

  attempt=1
  delay="$initial_delay"
  while true; do
    echo "Publishing $vsix to $registry (attempt $attempt/$max_attempts)"
    output_file="$(mktemp)"

    set +e
    publish_one "$vsix" 2>&1 | tee "$output_file"
    status="${PIPESTATUS[0]}"
    set -e

    if (( status == 0 )); then
      rm -f "$output_file"
      break
    fi

    if grep -Eqi "$already_published_pattern" "$output_file"; then
      echo "$vsix is already published; treating the operation as successful."
      rm -f "$output_file"
      break
    fi

    if (( attempt >= max_attempts )) || ! grep -Eqi "$transient_pattern" "$output_file"; then
      rm -f "$output_file"
      exit "$status"
    fi

    rm -f "$output_file"
    echo "Transient publish failure; retrying in ${delay}s."
    sleep "$delay"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done

  if (( index + 1 < ${#vsixes[@]} && between_files > 0 )); then
    echo "Pacing platform publishes for ${between_files}s."
    sleep "$between_files"
  fi
done
