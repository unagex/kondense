# How is container CPU calculated ?
Kondense resize each container CPU every second with the following steps.

## 1. Calculate average CPU usage of the container.
Kondense calculate the average CPU usage over the last `INTERVAL` seconds. By default, `INTERVAL` is 10 seconds.

### 2. Calculate new CPU
To calculate the new container CPU, we need the `TARGET_AVG`. By default, `TARGET_AVG` is 0.8 so 80%.

```
new_cpu = average_cpu / TARGET_AVG
```

### 2.1 If `new_cpu` is smaller than current container cpu limit

If `new_cpu` is smaller than the current container limit, we just patch container CPU limit with this `new_cpu`.

### 2.2 If `new_cpu` is bigger than current container cpu limit

If this `new_cpu` is higher than the current CPU container limit, we exponentially increase `new_cpu` with this formula:
```
new_cpu = new_cpu + (new_cpu * coeff)²
```
The bigger the `coeff`, the stronger Kondense will increase the CPU.
We patch the container CPU limit with this `new_cpu`.