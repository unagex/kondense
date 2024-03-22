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

		interval := DefaultMemoryInterval
		if v, ok := pod.Annotations[fmt.Sprintf("unagex.com/kondense-%s-memory-interval", containerStatus.Name)]; ok {
			var err error
			interval, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				r.L.Printf("error cannot parse memory interval in annotations for container: %s. Set memory interval to default value: %d.",
					containerStatus.Name, DefaultMemoryInterval)
				interval = DefaultMemoryInterval
			}
		}

		if _, ok := r.CStats[containerStatus.Name]; !ok {
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{GraceTicks: interval, Interval: interval}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}
