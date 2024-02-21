# Kondense

1. Start minikube
```bash
minikube start --kubernetes-version=v1.27.0 --feature-gates=InPlacePodVerticalScaling=true
```

2. Create pod
```bash
kubectl apply -f pod.yaml
```

3. Patch Pod
```bash
kubectl patch pod nginx-sample --patch '{"spec":{"containers":[{"name":"nginx", "resources":{"limits":{"memory": "300Mi", "cpu":"0.3"},"requests":{"memory": "300Mi", "cpu":"0.3"}}}]}}'
```