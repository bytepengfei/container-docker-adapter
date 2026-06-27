#!/bin/sh
set -eu

if [ "$#" -ne 4 ]; then
  echo "usage: $0 VERSION SOURCE_URL SHA256 OUTPUT" >&2
  exit 2
fi

version=$1
source_url=$2
sha256=$3
output=$4
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
template="$root/packaging/homebrew/container-docker-adapter.rb.tmpl"

sed \
  -e "s|__VERSION__|$version|g" \
  -e "s|__SOURCE_URL__|$source_url|g" \
  -e "s|__SHA256__|$sha256|g" \
  "$template" >"$output"
