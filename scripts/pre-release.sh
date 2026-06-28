#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

tag=$1
case "$tag" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *)
    echo "release tag must look like v0.1.0" >&2
    exit 2
    ;;
esac

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

if ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git status --short)" ]; then
  echo "working tree must be clean before release validation" >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  echo "tag already exists: $tag" >&2
  exit 1
fi

container system status >/dev/null
go test ./...
go vet ./...
APPLE_CONTAINER_E2E=1 ./scripts/e2e-apple.sh

echo "Pre-release validation passed for $tag"
echo "Publish with:"
echo "  git tag -a $tag -m \"$tag\""
echo "  git push origin $tag"
