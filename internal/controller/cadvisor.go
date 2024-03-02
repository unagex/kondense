package controller

import (
	"fmt"
	"strings"
	"time"

	cadvisorinfo "github.com/google/cadvisor/info/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Resources struct {
	memoryUsage uint64
	cpuUsage    uint64
}

type NamedResources = map[string]Resources

func (r *Reconciler) GetCadvisorData(pod *corev1.Pod) (NamedResources, ctrl.Result, error) {
	toExclude := []string{}
	l, ok := pod.Annotations["app.kubernetes.io/resources-managed-exclude"]
	if ok {
		toExclude = strings.Split(l, ",")
	}

	if len(pod.Status.ContainerStatuses) != len(pod.Spec.Containers) {
		return NamedResources{}, ctrl.Result{RequeueAfter: time.Second, Requeue: true}, nil
	}

	ress := NamedResources{}
	for _, cStat := range pod.Status.ContainerStatuses {
		if slices.Contains(toExclude, cStat.Name) {
			continue
		}

		if cStat.ContainerID == "" {
			// ContainerID can make some time to be populated, we requeue if it's
			// not the case.
			return NamedResources{}, ctrl.Result{RequeueAfter: 1 * time.Second, Requeue: true}, nil
		}

		if !strings.HasPrefix(cStat.ContainerID, "docker://") {
			return NamedResources{}, ctrl.Result{}, fmt.Errorf("docker is the only container runtime allowed")
		}
		trimmedContainerID := strings.TrimPrefix(cStat.ContainerID, "docker://")
		cInfos, err := r.Cclient.Stats(trimmedContainerID, &cadvisorinfo.RequestOptions{
			Recursive: false,
			IdType:    cadvisorinfo.TypeDocker,
			Count:     1,
		})
		if err != nil {
			return NamedResources{}, ctrl.Result{}, err
		}

		if len(cInfos) != 1 {
			return NamedResources{}, ctrl.Result{}, fmt.Errorf("should get info on only one container, got: %d", len(cInfos))
		}

		for _, cInfo := range cInfos {
			memoryUsage := cInfo.Stats[0].Memory.Usage
			cpuUsage := cInfo.Stats[0].Cpu.Usage.Total

			ress[cStat.Name] = Resources{
				memoryUsage: memoryUsage,
				cpuUsage:    cpuUsage,
			}
		}
	}

	return ress, ctrl.Result{}, nil
}
