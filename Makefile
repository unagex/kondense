VERSION ?= test
IMG ?= ghcr.io/unagex/kondense:${VERSION}

all: build

build:
	docker build -t ${IMG} .

deploy:
	minikube image load ${IMG}
	kubectl apply -f manifests

undeploy:
	kubectl delete -f manifests