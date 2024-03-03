# Kondense

## Requirements
kubernetes >=v1.27
containerd >=v1.6.9

1. Start kind with the feature gate InPlacePodVerticalScaling.
```bash
kind create cluster --config=dev/kind-config.yaml
```

2. Create pod
```bash
kubectl apply -f pod.yaml
```

3. Patch Pod
```bash
kubectl patch pod memory-test --patch '{"spec":{"containers":[{"name":"ubuntu", "resources":{"limits":{"memory": "200Mi", "cpu":"100m"},"requests":{"memory": "200Mi", "cpu":"100m"}}}]}}'
```

4. Scaleway add feature gate
```bash
scw k8s cluster update 0b4db211-543d-407e-9d3e-e3c7b9945fe5 feature-gates.0=InPlacePodVerticalScaling
```

5. Install cAdvisor
```bash
kubectl apply -f cadvisor
```

6. cAdvisor API
http://localhost:8080/api/v2.1/stats/<docker container id or name in docker ps>?type=docker&count=1
http://127.0.0.1:51464/api/v2.1/summary/<docker container is or name in docker ps>?type=docker&count=1