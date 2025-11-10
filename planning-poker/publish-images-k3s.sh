#!/bin/bash

## Call it like

## DOCKER_USERNAME=<username> ./publish-images-k3s.sh <tag>

set -e

IMAGE_TAG=${1:-"latest.k3s"}

if [ -z "$DOCKER_USERNAME" ]; then
  echo "DOCKER_USERNAME must be set"
  exit 1
fi

if [[ "$IMAGE_TAG" != *.k3s ]]; then
  IMAGE_TAG="${IMAGE_TAG}.k3s"
fi

make docker-build-frontend BACKEND_URL="https://planning-poker-backend.bsoapp.net" WEBSOCKET_URL="wss://planning-poker-backend.bsoapp.net" &

docker tag planning-poker-frontend $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG &

docker push $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG
