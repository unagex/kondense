package controller

import (
	"fmt"
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (r Reconciler) InitCStats(pod *corev1.Pod) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		exclude := containersToExclude(pod)
		if slices.Contains(exclude, containerStatus.Name) {
			continue
		}

		if _, ok := r.CStats[containerStatus.Name]; !ok {
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{
					Min:            r.getMemoryMin(pod, containerStatus.Name),
					Max:            r.getMemoryMax(pod, containerStatus.Name),
					GraceTicks:     r.getMemoryInterval(pod, containerStatus.Name),
					Interval:       r.getMemoryInterval(pod, containerStatus.Name),
					TargetPressure: r.getMemoryTargetPressure(pod, containerStatus.Name),
					MaxProbe:       r.getMemoryMaxProbe(pod, containerStatus.Name),
					MaxBackOff:     r.getMemoryMaxBackoff(pod, containerStatus.Name),
					CoeffProbe:     r.getMemoryCoeffProbe(pod, containerStatus.Name),
					CoeffBackoff:   r.getMemoryCoeffBackoff(pod, containerStatus.Name),
				}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}

func (r Reconciler) getMemoryMin(pod *corev1.Pod, containerName string) uint64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-min", containerName)]; ok {
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

func (r Reconciler) getMemoryMax(pod *corev1.Pod, containerName string) uint64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max", containerName)]; ok {
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

func (r Reconciler) getMemoryInterval(pod *corev1.Pod, containerName string) uint64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-interval", containerName)]; ok {
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

func (r Reconciler) getMemoryTargetPressure(pod *corev1.Pod, containerName string) uint64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-target-pressure", containerName)]; ok {
		targetPressure, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory target pressure in annotations for container: %s. Set memory target pressure to default value: %d.",
				containerName, DefaultMemTargetPressure)
			return DefaultMemTargetPressure
		}
		return targetPressure
	}

	return DefaultMemTargetPressure
}

func (r Reconciler) getMemoryMaxProbe(pod *corev1.Pod, containerName string) float64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max-probe", containerName)]; ok {
		maxProbe, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory max probe in annotations for container: %s. Set memory max probe to default value: %.2f.",
				containerName, DefaultMemMaxProbe)
			return DefaultMemMaxProbe
		}
		if maxProbe <= 0 || maxProbe >= 1 {
			r.L.Printf("error memory max probe in annotations should be between 0 and 1 exclusive for container: %s. Set memory max probe to default value: %.2f.",
				containerName, DefaultMemMaxProbe)
			return DefaultMemMaxProbe
		}
		return maxProbe
	}

	return DefaultMemMaxProbe
}

func (r Reconciler) getMemoryMaxBackoff(pod *corev1.Pod, containerName string) float64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max-backoff", containerName)]; ok {
		maxBackoff, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory max backoff in annotations for container: %s. Set memory max backoff to default value: %.2f.",
				containerName, DefaultMemMaxBackoff)
			return DefaultMemMaxBackoff
		}
		if maxBackoff <= 0 {
			r.L.Printf("error memory max backoff in annotations should be bigger than 0 for container: %s. Set memory max backoff to default value: %.2f.",
				containerName, DefaultMemMaxBackoff)
			return DefaultMemMaxBackoff
		}
		return maxBackoff
	}

	return DefaultMemMaxBackoff
}

func (r Reconciler) getMemoryCoeffProbe(pod *corev1.Pod, containerName string) float64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-coeff-probe", containerName)]; ok {
		coeffProbe, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory coeff probe in annotations for container: %s. Set memory coeff probe to default value: %.2f.",
				containerName, DefaultMemCoeffProbe)
			return DefaultMemCoeffProbe
		}
		if coeffProbe <= 0 {
			r.L.Printf("error memory coeff probe in annotations should be bigger than 0 for container: %s. Set memory coeff probe to default value: %.2f.",
				containerName, DefaultMemCoeffProbe)
			return DefaultMemCoeffProbe
		}
		return coeffProbe
	}

	return DefaultMemCoeffProbe
}

func (r Reconciler) getMemoryCoeffBackoff(pod *corev1.Pod, containerName string) float64 {
	if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-coeff-backoff", containerName)]; ok {
		coeffBackoff, err := strconv.ParseFloat(v, 64)
		if err != nil {
			r.L.Printf("error cannot parse memory coeff backoff in annotations for container: %s. Set memory coeff backoff to default value: %.2f.",
				containerName, DefaultMemCoeffBackoff)
			return DefaultMemCoeffBackoff
		}
		if coeffBackoff <= 0 {
			r.L.Printf("error memory coeff backoff in annotations should be bigger than 0 for container: %s. Set memory coeff backoff to default value: %.2f.",
				containerName, DefaultMemCoeffBackoff)
			return DefaultMemCoeffBackoff
		}
		return coeffBackoff
	}

	return DefaultMemCoeffBackoff
}
