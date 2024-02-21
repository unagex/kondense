# Kondense

## Requirements
kubernetes >=v1.27
containerd >=v1.6.9

1. Start minikube with the feature gate InPlacePodVerticalScaling.
```bash
minikube start --kubernetes-version=v1.29.2 --feature-gates=InPlacePodVerticalScaling=true
```

2. Create pod
```bash
kubectl apply -f pod.yaml
```

3. Patch Pod
```bash
kubectl patch pod nginx-sample --patch '{"spec":{"containers":[{"name":"nginx", "resources":{"limits":{"memory": "300Mi", "cpu":"0.3"},"requests":{"memory": "300Mi", "cpu":"0.3"}}}]}}'
```