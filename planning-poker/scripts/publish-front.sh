#!/bin/bash

## Call it like

## DOCKER_USERNAME=<username> ./publish-front.sh <tag>

set -e

IMAGE_TAG=${1:-"latest"}

if [ -z "$DOCKER_USERNAME" ]; then
  echo "DOCKER_USERNAME must be set"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

make docker-build-frontend BACKEND_URL="https://planning-poker-backend.bsoapp.net" WEBSOCKET_URL="wss://planning-poker-backend.bsoapp.net"
docker tag planning-poker-frontend $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG
docker push $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG
