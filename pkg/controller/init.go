package controller

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (r *Reconciler) InitCStats(pod *corev1.Pod) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		exclude := containersToExclude()
		if slices.Contains(exclude, containerStatus.Name) {
			continue
		}

		if _, ok := r.CStats[containerStatus.Name]; !ok {
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{
					Min:            r.getMemoryMin(containerStatus.Name),
					Max:            r.getMemoryMax(containerStatus.Name),
					GraceTicks:     r.getMemoryInterval(containerStatus.Name),
					Interval:       r.getMemoryInterval(containerStatus.Name),
					TargetPressure: r.getMemoryTargetPressure(containerStatus.Name),
					MaxInc:         r.getMemoryMaxInc(containerStatus.Name),
					MaxDec:         r.getMemoryMaxDec(containerStatus.Name),
					CoeffInc:       r.getMemoryCoeffInc(containerStatus.Name),
					CoeffDec:       r.getMemoryCoeffDec(containerStatus.Name),
				}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}

func (r *Reconciler) getMemoryMin(containerName string) uint64 {
	env := fmt.Sprintf("%s_MEMORY_MIN", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		minQ, err := resource.ParseQuantity(v)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %d bytes.",
				env, env, DefaultMemMin)
			return DefaultMemMin
		}
		min := minQ.Value()
		if min <= 0 {
			r.L.Printf("error environment variable: %s should be bigger than 0. Set %s to default value: %d bytes",
				env, env, DefaultMemMin)
			return DefaultMemMin
		}
		return uint64(min)
	}

	return DefaultMemMin
}

func (r *Reconciler) getMemoryMax(containerName string) uint64 {
	env := fmt.Sprintf("%s_MEMORY_MAX", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		maxQ, err := resource.ParseQuantity(v)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %d bytes.",
				env, env, DefaultMemMax)
			return DefaultMemMax
		}
		max := maxQ.Value()
		if max <= 0 {
			r.L.Printf("error environment variable: %s should be bigger than 0. Set %s to default value: %d bytes",
				env, env, DefaultMemMax)
			return DefaultMemMax
		}
		return uint64(max)
	}

	return DefaultMemMax
}

func (r *Reconciler) getMemoryInterval(containerName string) uint64 {
	env := fmt.Sprintf("%s_MEMORY_INTERVAL", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		interval, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %d.",
				env, env, DefaultMemInterval)
			return DefaultMemInterval
		}
		return interval
	}

	return DefaultMemInterval
}

func (r *Reconciler) getMemoryTargetPressure(containerName string) uint64 {
	env := fmt.Sprintf("%s_MEMORY_TARGET_PRESSURE", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		targetPressure, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s pressure. Set %s to default value: %d.",
				env, env, DefaultMemTargetPressure)
			return DefaultMemTargetPressure
		}
		if targetPressure == 0 {
			r.L.Printf("error environment variable: %s should be more than 0. Set %s to default value: %d.",
				env, env, DefaultMemTargetPressure)
			return DefaultMemTargetPressure
		}
		return targetPressure
	}

	return DefaultMemTargetPressure
}

func (r *Reconciler) getMemoryMaxDec(containerName string) float64 {
	env := fmt.Sprintf("%s_MEMORY_MAX_DEC", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		maxDec, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %.2f.",
				env, env, DefaultMemMaxDec)
			return DefaultMemMaxDec
		}
		if maxDec <= 0 || maxDec >= 1 {
			r.L.Printf("error environment variable: %s should be between 0 and 1 exclusive. Set %s to default value: %.2f.",
				env, env, DefaultMemMaxDec)
			return DefaultMemMaxDec
		}
		return maxDec
	}

	return DefaultMemMaxDec
}

func (r *Reconciler) getMemoryMaxInc(containerName string) float64 {
	env := fmt.Sprintf("%s_MEMORY_MAX_INC", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		maxInc, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %.2f.",
				env, env, DefaultMemMaxInc)
			return DefaultMemMaxInc
		}
		if maxInc <= 0 {
			r.L.Printf("error environment variable: %s should be bigger than 0. Set %s to default value: %.2f.",
				env, env, DefaultMemMaxInc)
			return DefaultMemMaxInc
		}
		return maxInc
	}

	return DefaultMemMaxInc
}

func (r *Reconciler) getMemoryCoeffDec(containerName string) float64 {
	env := fmt.Sprintf("%s_MEMORY_COEFF_DEC", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		coeffDec, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %.2f.",
				env, env, DefaultMemCoeffDec)
			return DefaultMemCoeffDec
		}
		if coeffDec <= 0 {
			r.L.Printf("error environment variable: %s should be bigger than 0. Set %s to default value: %.2f.",
				env, env, DefaultMemCoeffDec)
			return DefaultMemCoeffDec
		}
		return coeffDec
	}

	return DefaultMemCoeffDec
}

func (r *Reconciler) getMemoryCoeffInc(containerName string) float64 {
	env := fmt.Sprintf("%s_MEMORY_MAX_INC", strings.ToUpper(containerName))
	if v, ok := os.LookupEnv(env); ok {
		coeffInc, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse environment variable: %s. Set %s to default value: %.2f.",
				env, env, DefaultMemCoeffInc)
			return DefaultMemCoeffInc
		}
		if coeffInc <= 0 {
			r.L.Printf("error environment variable: %s should be bigger than 0. Set %s to default value: %.2f.",
				env, env, DefaultMemCoeffInc)
			return DefaultMemCoeffInc
		}
		return coeffInc
	}

	return DefaultMemCoeffInc
}
