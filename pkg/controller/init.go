package controller

import (
	"fmt"
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

func (r Reconciler) InitCStats(pod *corev1.Pod) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		exclude := containersToExclude(pod)
		if slices.Contains(exclude, containerStatus.Name) {
			continue
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

		if _, ok := r.CStats[containerStatus.Name]; !ok {
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{
					GraceTicks:     interval,
					Interval:       interval,
					TargetPressure: targetPressure,
					MaxProbe:       maxProbe,
				}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}
