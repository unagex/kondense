IMG ?= kondense/kondense:1.1.0

all: build

build:
	docker build -t ${IMG} .

load:
	minikube image load ${IMG}

deploy:
	kubectl apply -f manifests

undeploy:
	kubectl delete -f manifests

push:
	docker push ${IMG}