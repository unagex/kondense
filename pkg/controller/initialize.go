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
		if _, ok := r.Res[containerStatus.Name]; !ok {
			// GraceTicks and Interval default to 6.
			r.Res[containerStatus.Name] = &Stats{
				Mem: Memory{GraceTicks: 6, Interval: 6}}
		}

		limit := containerStatus.AllocatedResources.Memory().Value()
		r.Res[containerStatus.Name].Mem.Limit = limit
	}
}
