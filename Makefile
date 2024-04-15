IMG ?= kondense/kondense:1.0.1
GO_BUILD_FLAGS ?=

# Enable Docker BuildKit
export DOCKER_BUILDKIT=1

all: build

build:
	docker build --build-arg GO_BUILD_FLAGS="${GO_BUILD_FLAGS}" -t ${IMG} .

load:
	minikube image load ${IMG}

deploy:
	kubectl apply -f manifests

undeploy:
	kubectl delete -f manifests

push:
	docker push ${IMG}

.PHONY: all build load deploy undeploy push