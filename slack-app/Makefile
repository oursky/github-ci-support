IMAGE_REPO=ghcr.io/oursky/github-ci-support/slack-app
GIT_COMMIT=$(shell git rev-parse HEAD)\
TAG?=latest

.PHONY: build
build:
	docker buildx build . -t "${IMAGE_REPO}:build" ${BUILDX_ARGS}

.PHONY: push
push:
	docker image tag "${IMAGE_REPO}:build" "${IMAGE_REPO}:${TAG}"
	docker image tag "${IMAGE_REPO}:build" "${IMAGE_REPO}:sha-$$(git rev-parse --short=10 HEAD)"
	docker push "${IMAGE_REPO}:${TAG}"
	docker push "${IMAGE_REPO}:sha-$$(git rev-parse --short=10 HEAD)"
