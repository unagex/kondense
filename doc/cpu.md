# How is container CPU calculated ?
Kondense resize each container CPU every second with the following steps.

## 1. Calculate average CPU usage
Kondense calculate the average CPU usage over the last `INTERVAL` seconds. By default, `INTERVAL` is 6 seconds.

### 2. Calculate new CPU
We need the `TARGET_AVG` to calculate the new CPU. By default, `TARGET_AVG` is 0.8 so 80%.

```
new_cpu = average_cpu / TARGET_AVG
```

### 2.1 If `new_cpu` is smaller than current CPU limit

If `new_cpu` is smaller than the current CPU limit, we just patch CPU limit with this `new_cpu`.

### 2.2 If `new_cpu` is bigger than current CPU limit

If `new_cpu` is higher than the current CPU limit, we exponentially increase `new_cpu` with this formula:
```
new_cpu = new_cpu + (new_cpu * coeff)²
```
The bigger the `coeff`, the stronger Kondense will increase the CPU.
We patch the CPU limit with this `new_cpu`.