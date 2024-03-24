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

func (r Reconciler) InitCStats(pod *corev1.Pod) {
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

func (r Reconciler) getMemoryMin(containerName string) uint64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_MIN", strings.ToUpper(containerName))); ok {
		minQ, err := resource.ParseQuantity(v)
		if err != nil {
			r.L.Printf("error cannot parse memory minimum in annotations for container: %s. Set memory minimum to default value: %d bytes.",
				containerName, DefaultMemMin)
			return DefaultMemMin
		}
		min := minQ.Value()
		if min <= 0 {
			r.L.Printf("error memory minimum in annotations should be bigger than 0 for container: %s. Set memory minimum to default value: %d bytes",
				containerName, DefaultMemMin)
			return DefaultMemMin
		}
		return uint64(min)
	}

	return DefaultMemMin
}

func (r Reconciler) getMemoryMax(containerName string) uint64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_MAX", strings.ToUpper(containerName))); ok {
		maxQ, err := resource.ParseQuantity(v)
		if err != nil {
			r.L.Printf("error cannot parse memory maximum in annotations for container: %s. Set memory maximum to default value: %d bytes.",
				containerName, DefaultMemMax)
			return DefaultMemMax
		}
		max := maxQ.Value()
		if max <= 0 {
			r.L.Printf("error memory maximum in annotations should be bigger than 0 for container: %s. Set memory maximum to default value: %d bytes",
				containerName, DefaultMemMax)
			return DefaultMemMax
		}
		return uint64(max)
	}

	return DefaultMemMax
}

func (r Reconciler) getMemoryInterval(containerName string) uint64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_INTERVAL", strings.ToUpper(containerName))); ok {
		interval, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory interval in annotations for container: %s. Set memory interval to default value: %d.",
				containerName, DefaultMemInterval)
			return DefaultMemInterval
		}
		return interval
	}

	return DefaultMemInterval
}

func (r Reconciler) getMemoryTargetPressure(containerName string) uint64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_TARGET_PRESSURE", strings.ToUpper(containerName))); ok {
		targetPressure, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory target pressure in annotations for container: %s. Set memory target pressure to default value: %d.",
				containerName, DefaultMemTargetPressure)
			return DefaultMemTargetPressure
		}
		if targetPressure == 0 {
			r.L.Printf("error memory target pressure in annotations should be more than 0 for container: %s. Set memory target pressure to default value: %d.",
				containerName, DefaultMemTargetPressure)
			return DefaultMemTargetPressure
		}
		return targetPressure
	}

	return DefaultMemTargetPressure
}

func (r Reconciler) getMemoryMaxDec(containerName string) float64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_MAX_DEC", strings.ToUpper(containerName))); ok {
		maxDec, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memoryMaxDec in annotations for container: %s. Set memoryMaxDec to default value: %.2f.",
				containerName, DefaultMemMaxDec)
			return DefaultMemMaxDec
		}
		if maxDec <= 0 || maxDec >= 1 {
			r.L.Printf("error memoryMaxDec in annotations should be between 0 and 1 exclusive for container: %s. Set memoryMaxDec to default value: %.2f.",
				containerName, DefaultMemMaxDec)
			return DefaultMemMaxDec
		}
		return maxDec
	}

	return DefaultMemMaxDec
}

func (r Reconciler) getMemoryMaxInc(containerName string) float64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_MAX_INC", strings.ToUpper(containerName))); ok {
		maxInc, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memoryMaxInc in annotations for container: %s. Set memoryMaxInc to default value: %.2f.",
				containerName, DefaultMemMaxInc)
			return DefaultMemMaxInc
		}
		if maxInc <= 0 {
			r.L.Printf("error memoryMaxInc in annotations should be bigger than 0 for container: %s. Set memoryMaxInc to default value: %.2f.",
				containerName, DefaultMemMaxInc)
			return DefaultMemMaxInc
		}
		return maxInc
	}

	return DefaultMemMaxInc
}

func (r Reconciler) getMemoryCoeffDec(containerName string) float64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_COEFF_DEC", strings.ToUpper(containerName))); ok {
		coeffDec, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memoryCoeffDec in annotations for container: %s. Set memoryCoeffDec to default value: %.2f.",
				containerName, DefaultMemCoeffDec)
			return DefaultMemCoeffDec
		}
		if coeffDec <= 0 {
			r.L.Printf("error memoryCoeffDec in annotations should be bigger than 0 for container: %s. Set memoryCoeffDec to default value: %.2f.",
				containerName, DefaultMemCoeffDec)
			return DefaultMemCoeffDec
		}
		return coeffDec
	}

	return DefaultMemCoeffDec
}

func (r Reconciler) getMemoryCoeffInc(containerName string) float64 {
	if v, ok := os.LookupEnv(fmt.Sprintf("%s_MEMORY_COEFF_INC", strings.ToUpper(containerName))); ok {
		coeffInc, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memoryCoeffInc in annotations for container: %s. Set memoryCoeffInc to default value: %.2f.",
				containerName, DefaultMemCoeffInc)
			return DefaultMemCoeffInc
		}
		if coeffInc <= 0 {
			r.L.Printf("error memoryCoeffInc in annotations should be bigger than 0 for container: %s. Set memoryCoeffInc to default value: %.2f.",
				containerName, DefaultMemCoeffInc)
			return DefaultMemCoeffInc
		}
		return coeffInc
	}

	return DefaultMemCoeffInc
}
