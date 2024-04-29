IMG ?= kondense/kondense:1.1.0

all: build

build:
	docker build -t ${IMG} .

load:
	minikube image load ${IMG}

deploy:
	kubectl apply -f dev

undeploy:
	kubectl delete -f dev

push:
	docker push ${IMG}