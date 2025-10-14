#!/bin/bash

## Call it like

## DOCKER_USERNAME=<username> ./publish-images.sh <tag>

set -e

IMAGE_TAG=${1:-"latest"}

if [ -z "$DOCKER_USERNAME" ]; then
  echo "DOCKER_USERNAME must be set"
  exit 1
fi

make docker-build-frontend BACKEND_URL="https://planning-poker-backend.bsoapp.net" WEBSOCKET_URL="wss://planning-poker-backend.bsoapp.net" &
make docker-build-backend OTLP_ENDPOINT="tempo:4317" &
wait

docker tag planning-poker-backend $DOCKER_USERNAME/planning-poker-backend:$IMAGE_TAG &
docker tag planning-poker-frontend $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG &
wait

docker push $DOCKER_USERNAME/planning-poker-backend:$IMAGE_TAG
docker push $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG
