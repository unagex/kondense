# Kondense

![Go version](https://img.shields.io/github/go-mod/go-version/unagex/kondense)
[![Go Report Card](https://goreportcard.com/badge/github.com/unagex/kondense)](https://goreportcard.com/report/github.com/unagex/kondense)
[![GitHub release](https://img.shields.io/github/v/tag/unagex/kondense.svg?label=release&color=lightgrey)](https://github.com/unagex/kondense/releases)

<img src="./logo.png" alt="drawing" width="150"/>

Kondense is an automated resource sizing tool. It runs as a sidecar in kubernetes pods.

## Background

### Memory
Kondense uses memory pressure to apply just the right amount of memory on a container to page out the unused memory while not getting out-of-memory killed.

### CPU
Kondense resizes CPU based on CPU usage, default to 80%.

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
Try the example on minikube:
```bash
minikube start --kubernetes-version=v1.29.2 --feature-gates=InPlacePodVerticalScaling=true
kubectl apply -f https://raw.githubusercontent.com/unagex/kondense/main/example/nginx.yaml
```

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
        cpu: 80M
        memory: 50M
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
    image: kondense/kondense:1.1.0
    resources:
      limits:
        cpu: 80m
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

After adding the kondense container, the nginx container resources will be updated dynamically without any container restart.

## Configuration

Kondense is configurable via environment variables in the kondense container.

#### Example
```yaml
    ...
    - name: kondense
      image: kondense/kondense:1.1.0
      resources:
        limits:
          cpu: 80m
          memory: 50M
      env:
      - name: <CONTAINER NAME>_MEMORY_MIN
        value: "100m"
```

If we have a container named `nginx` in our pod, the variable name should be `NGINX_MEMORY_MIN`.

### Environment variables
#### Global

| Name | Default value | Description |
| --- | --- | --- |
| EXCLUDE | "" | Comma separated list of containers to not kondense. |

#### Memory
| Name | Default value | Description |
| --- | --- | --- |
| \<CONTAINER NAME>\_MEMORY_MIN | 50M | Minimum memory of the container. Kondense will never resize below that limit. |
| \<CONTAINER NAME>\_MEMORY_MAX | 100G | Maximum memory of the container. Kondense will never resize above that limit. |
| \<CONTAINER NAME>\_MEMORY_TARGET_PRESSURE | 10000 | Target memory pressure in microseconds. Kondense will take corrective actions to obtain it. |
| \<CONTAINER NAME>\_MEMORY_INTERVAL | 10 | Kondense targets cumulative memory delays over the sampling period of this interval in seconds. |
| \<CONTAINER NAME>\_MEMORY_MAX_INC | 0.5 | Maximum memory increase for one correction. e.g. 0.5 is a 50% increase. |
| \<CONTAINER NAME>\_MEMORY_MAX_DEC | 0.02 | Maximum memory decrease for one correction. e.g. 0.02 is a 2% decrease. |
| \<CONTAINER NAME>\_MEMORY_COEFF_INC | 20 | Coeff to increase memory  when the memory pressure is bigger then the target memory pressure. |
| \<CONTAINER NAME>\_MEMORY_COEFF_DEC | 10 | Coeff to decrease memory when the memory pressure is smaller then the target memory pressure. |

#### CPU
| Name | Default value | Description |
| --- | --- | --- |
| \<CONTAINER NAME>\_CPU_MIN | 0.08 | Minimum CPU of the container. Kondense will never resize below that limit. |
| \<CONTAINER NAME>\_CPU_MAX | 100 | Maximum CPU of the container. Kondense will never resize above that limit. |
| \<CONTAINER NAME>\_CPU_MAX_INC | 0.5 | Maximum CPU increase for one correction. e.g. 0.5 is a 50% increase. |
| \<CONTAINER NAME>\_CPU_MAX_DEC | 0.1 | Maximum CPU decrease for one correction. e.g. 0.1 is a 10% decrease. |
| \<CONTAINER NAME>\_CPU_TARGET_AVG | 0.8 | Target CPU average for the container. It is from 0 to 1. e.g. 0.8 means a target cpu usage of 80%. |
| \<CONTAINER NAME>\_CPU_INTERVAL | 6 | Target CPU average for the container. It is from 0 to 1. e.g. 0.8 means a target cpu usage of 80%. |
| \<CONTAINER NAME>\_CPU_COEFF | 6 | Used to calculate the new cpu limit when a cpu increase is needed. The higher the coeff, the higher the new cpu limit. |

### More
- Kondense memory resize is based on [Facebook senpai](https://github.com/facebookincubator/senpai/tree/main)
- Kondense is active on himself by default