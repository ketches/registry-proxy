DOCKER_IMAGE := ketches/registry-proxy
ALIYUN_IMAGE := registry.cn-hangzhou.aliyuncs.com/ketches/registry-proxy
VERSION := v1.3.1

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/amd64/registry-proxy main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/arm64/registry-proxy main.go
	docker buildx create --use --name gobuilder 2>/dev/null || docker buildx use gobuilder
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):$(VERSION) -t $(ALIYUN_IMAGE):$(VERSION) --push . -f Dockerfile.local

.PHONY: deploy
deploy:
	kubectl apply -f deploy/manifests.yaml

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/manifests.yaml