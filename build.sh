#!/bin/sh
set -ev

echo "Build & push"
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
docker build . -t snimmagadda/github-pr-bot:"$TRAVIS_BUILD_NUMBER"
docker tag snimmagadda/github-pr-bot:"$TRAVIS_BUILD_NUMBER" snimmagadda/github-pr-bot:latest
docker push snimmagadda/github-pr-bot:"$TRAVIS_BUILD_NUMBER" && docker push snimmagadda/github-pr-bot:latest