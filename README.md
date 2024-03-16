# Kondense

## Requirements
kubernetes >=v1.27
containerd >=v1.6.9

1. Start kind with the feature gate InPlacePodVerticalScaling.
```bash
minikube start --kubernetes-version=v1.29.2 --feature-gates=InPlacePodVerticalScaling=true
```

2. Create pod
```bash
kubectl apply -f pod.yaml
```

3. Patch Pod
```bash
kubectl patch pod resources-test --patch '{"spec":{"containers":[{"name":"ubuntu", "resources":{"limits":{"memory": "200Mi", "cpu":"100m"},"requests":{"memory": "200Mi", "cpu":"100m"}}}]}}'
```

4. Scaleway add feature gate
```bash
scw k8s cluster update 0b4db211-543d-407e-9d3e-e3c7b9945fe5 feature-gates.0=InPlacePodVerticalScaling
```