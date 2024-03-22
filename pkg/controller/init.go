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

		min := DefaultMemMin
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-min", containerStatus.Name)]; ok {
			minQ, err := resource.ParseQuantity(v)
			minTmp := minQ.Value()
			min = uint64(minTmp)
			if err != nil {
				r.L.Printf("error cannot parse memory minimum in annotations for container: %s. Set memory minimum to default value: %d bytes.",
					containerStatus.Name, DefaultMemMin)
				min = DefaultMemMin
			}
			if minTmp <= 0 {
				r.L.Printf("error memory minimum in annotations should be bigger than 0 for container: %s. Set memory minimum to default value: %d bytes",
					containerStatus.Name, DefaultMemMin)
				min = DefaultMemMin
			}
		}

		max := DefaultMemMax
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max", containerStatus.Name)]; ok {
			maxQ, err := resource.ParseQuantity(v)
			maxTmp := maxQ.Value()
			max = uint64(maxTmp)
			if err != nil {
				r.L.Printf("error cannot parse memory maximum in annotations for container: %s. Set memory maximum to default value: %d bytes.",
					containerStatus.Name, DefaultMemMax)
				max = DefaultMemMax
			}
			if maxTmp <= 0 {
				r.L.Printf("error memory maximum in annotations should be bigger than 0 for container: %s. Set memory maximum to default value: %d bytes",
					containerStatus.Name, DefaultMemMax)
				max = DefaultMemMax
			}
		}

		interval := DefaultMemInterval
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-interval", containerStatus.Name)]; ok {
			var err error
			interval, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory interval in annotations for container: %s. Set memory interval to default value: %d.",
					containerStatus.Name, DefaultMemInterval)
				interval = DefaultMemInterval
			}
		}

		targetPressure := DefaultMemTargetPressure
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-target-pressure", containerStatus.Name)]; ok {
			var err error
			targetPressure, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory target pressure in annotations for container: %s. Set memory target pressure to default value: %d.",
					containerStatus.Name, DefaultMemTargetPressure)
				targetPressure = DefaultMemTargetPressure
			}
		}

		maxProbe := DefaultMemMaxProbe
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max-probe", containerStatus.Name)]; ok {
			var err error
			maxProbe, err = strconv.ParseFloat(v, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory max probe in annotations for container: %s. Set memory max probe to default value: %.2f.",
					containerStatus.Name, DefaultMemMaxProbe)
				maxProbe = DefaultMemMaxProbe
			}
			if maxProbe <= 0 || maxProbe >= 1 {
				r.L.Printf("error memory max probe in annotations should be between 0 and 1 exclusive for container: %s. Set memory max probe to default value: %.2f.",
					containerStatus.Name, DefaultMemMaxProbe)
				maxProbe = DefaultMemMaxProbe
			}
		}

		maxBackoff := DefaultMemMaxBackoff
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-max-backoff", containerStatus.Name)]; ok {
			var err error
			maxBackoff, err = strconv.ParseFloat(v, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory max backoff in annotations for container: %s. Set memory max backoff to default value: %.2f.",
					containerStatus.Name, DefaultMemMaxBackoff)
				maxBackoff = DefaultMemMaxBackoff
			}
			if maxBackoff <= 0 {
				r.L.Printf("error memory max backoff in annotations should be bigger than 0 for container: %s. Set memory max backoff to default value: %.2f.",
					containerStatus.Name, DefaultMemMaxBackoff)
				maxBackoff = DefaultMemMaxBackoff
			}
		}

		coeffProbe := DefaultMemCoeffProbe
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-coeff-probe", containerStatus.Name)]; ok {
			var err error
			coeffProbe, err = strconv.ParseFloat(v, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory coeff probe in annotations for container: %s. Set memory coeff probe to default value: %.2f.",
					containerStatus.Name, DefaultMemCoeffProbe)
				coeffProbe = DefaultMemCoeffProbe
			}
			if coeffProbe <= 0 {
				r.L.Printf("error memory coeff probe in annotations should be bigger than 0 for container: %s. Set memory coeff probe to default value: %.2f.",
					containerStatus.Name, DefaultMemCoeffProbe)
				coeffProbe = DefaultMemCoeffProbe
			}
		}

		coeffBackoff := DefaultMemCoeffBackoff
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-coeff-backoff", containerStatus.Name)]; ok {
			var err error
			coeffBackoff, err = strconv.ParseFloat(v, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory coeff backoff in annotations for container: %s. Set memory coeff backoff to default value: %.2f.",
					containerStatus.Name, DefaultMemCoeffBackoff)
				coeffBackoff = DefaultMemCoeffBackoff
			}
			if maxBackoff <= 0 {
				r.L.Printf("error memory coeff backoff in annotations should be bigger than 0 for container: %s. Set memory coeff backoff to default value: %.2f.",
					containerStatus.Name, DefaultMemCoeffBackoff)
				coeffBackoff = DefaultMemCoeffBackoff
			}
		}

		if _, ok := r.CStats[containerStatus.Name]; !ok {
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{
					Min:            min,
					Max:            max,
					GraceTicks:     interval,
					Interval:       interval,
					TargetPressure: targetPressure,
					MaxProbe:       maxProbe,
					MaxBackOff:     maxBackoff,
					CoeffBackoff:   coeffBackoff,
					CoeffProbe:     coeffProbe,
				}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}
