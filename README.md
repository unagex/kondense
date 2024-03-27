# Kondense
![Go version](https://img.shields.io/github/go-mod/go-version/unagex/kondense)
[![Go Report Card](https://goreportcard.com/badge/github.com/unagex/kondense)](https://goreportcard.com/report/github.com/unagex/kondense)

Kondense is an automated memory sizing tool. It runs as a sidecar in kubernetes pods.

## Background
Kondense uses real-time memory pressure to determine the optimal memory for each containers in a pod.

Allocated memory is not a good proxy for required memory. Many libraries used during startup are loaded into memory only to be never touched again afterwards. 

Kondense uses the memory pressure given by the Linux Kernel to apply just the right amount of memory on a container to page out the unused memory while not getting out-of-memory killed.

## Requirements

### On Kubernetes
1. The Kubernetes cluster must run on Linux.
2. Kubernetes version >= 1.27.
3. Containerd version >= 1.6.9.
4. Kubernetes should have the feature gate `InPlacePodVerticalScaling` enabled.

### On Containers
1. Containers should have the binary `cat`.
2. Containers should include the linux kernel version >= 4.20. Ensure the file `/sys/fs/cgroup/memory.pressure` exists in the container to verify it.

## Example

Let's say we have a pod running `nginx` that we want to Kondense:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kondense-test
spec:
  serviceAccountName: nginx-user
  containers:
  - name: nginx
    image: nginx:latest
    resources:
      limits:
        cpu: 0.1
        memory: 100M
```

Add Kondense as a sidecar:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kondense-test
spec:
  serviceAccountName: nginx-user
    containers:
    - name: nginx
      image: nginx:latest
      resources:
        limits:
          cpu: 0.1
          memory: 100M
    - name: kondense
      image: kondense/kondense:1.0.0
      resources:
        limits:
          cpu: 0.3
          memory: 50M
```

**Notes:**
1. The pod should have a QoS of `Guaranteed`. In other words, we need to add resources limits for each containers.
2. The service account `nginx-user` should have the following rules:
```yaml
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch", "patch"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
```

After adding the kondense container, the nginx container memory will be updated dynamically without any container restart.

## Configuration