.PHONY: update_gomod docker_build docker_push docker_release generate_cert

TAG ?= latest

update_gomod:
	@echo "Updating go modules..."
	@go get -u ./...
	@go mod tidy

docker_build:
	@echo "Building Docker image: resolyte/service-gps:$(TAG)"
	@docker build -f Dockerfile.gps -t resolyte/service-gps:$(TAG) .

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