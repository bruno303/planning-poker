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

make docker-build-frontend BACKEND_URL="https://k3s.bsoapp.net/planning-poker-backend" WEBSOCKET_URL="wss://k3s.bsoapp.net/planning-poker-backend" &
make docker-build-backend OTLP_ENDPOINT="tempo:4317" &
wait

docker tag planning-poker-backend $DOCKER_USERNAME/planning-poker-backend:$IMAGE_TAG &
docker tag planning-poker-frontend $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG &
wait

docker push $DOCKER_USERNAME/planning-poker-backend:$IMAGE_TAG
docker push $DOCKER_USERNAME/planning-poker-frontend:$IMAGE_TAG
