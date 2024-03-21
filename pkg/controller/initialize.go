package controller

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
)

func (r Reconciler) InitializeRes(pod *corev1.Pod) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		exclude := containersToExclude(pod)
		if slices.Contains(exclude, containerStatus.Name) {
			continue
		}

		// initialize container res if not already initialized
		if _, ok := r.CStats[containerStatus.Name]; !ok {
			// GraceTicks and Interval default to 2.
			r.CStats[containerStatus.Name] = &Stats{
				Mem: Memory{GraceTicks: 2, Interval: 2}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.CStats[containerStatus.Name].Mem.Limit = limit
	}
}
