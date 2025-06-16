.PHONY: update_gomod docker_build docker_push docker_release generate_cert

TAG ?= latest
GOPRIVATE := github.com/Resolyte-official/*
GOPROXY := direct

update_gomod:
	@if [ ! -f .github_token ]; then \
		echo ".github_token file not found"; exit 1; \
	fi
	@export GITHUB_TOKEN=$$(cat .github_token)
	@git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
	@echo "Updating go modules..."
	@GOPRIVATE=$(GOPRIVATE) GOPROXY=$(GOPROXY) go get -u ./...
	@GOPRIVATE=$(GOPRIVATE) GOPROXY=$(GOPROXY) go mod tidy

docker_build:
	@if [ ! -f .github_token ]; then \
		echo ".github_token file not found"; exit 1; \
	fi
	@echo "Building Docker image: resolyte/service-gps:$(TAG)"
	@DOCKER_BUILDKIT=1 docker build \
	--secret id=github_token,src=.github_token \
	-f Dockerfile.gps \
	-t resolyte/service-gps:$(TAG) .

docker_push:
	@echo "Pushing Docker image: resolyte/service-gps:$(TAG)"
	@docker push resolyte/service-gps:$(TAG)

docker_release: docker_build docker_push
	@echo "released $(TAG)"

generate_cert:
	@go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --ca true
	@mkdir -p configs
	@mv cert.pem key.pem ./configs/
	@echo "certs generated at configs/cert.pem and configs/key.pem"