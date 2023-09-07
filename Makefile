SHELL := /bin/bash

.PHONY: run build push deploy
run: main.go config.go config.yaml go.mod go.sum
	go run .

build: .tmp/image_built
.tmp/image_built: Dockerfile main.go config.go config.yaml go.mod go.sum
	docker buildx build --platform linux/amd64 -t petewall/test-workload .
	touch .tmp/image_built

push: .tmp/image_pushed
.tmp/image_pushed: .tmp/image_built
	docker push petewall/test-workload
	touch .tmp/image_pushed

deploy: .tmp/image_pushed
	kapp deploy -a test-workload --into-ns prod -f <(kbld -f deployment.yaml)
