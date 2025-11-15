#!/bin/bash

## Call it like

## DOCKER_USERNAME=<username> ./publish-images.sh <tag>

set -e

IMAGE_TAG=${1:-"latest"}

if [ -z "$DOCKER_USERNAME" ]; then
  echo "DOCKER_USERNAME must be set"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

./publish-back.sh $IMAGE_TAG
./publish-front.sh $IMAGE_TAG
