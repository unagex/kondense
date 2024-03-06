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

const Count = 5

type Resources struct {
	// current memory usage in bytes.
	memoryUsage uint64

	// average of cpu usage for the last <Count> timestamps.
	cpuUsage float64
	// cpuThrottled in percent of the cpu that is throttled.
	// e.g. cpuThrottled = 0.4 means we needed 40% more cpuUsage to have no throttling at all.
	// cpuThrottled float64
}

type NamedResources = map[string]Resources

func (r *Reconciler) GetCadvisorData(pod *corev1.Pod) (NamedResources, ctrl.Result, error) {
	toExclude := []string{}
	l, ok := pod.Annotations["unagex.com/resources-managed-exclude"]
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
			Count:     Count,
		})
		if err != nil {
			return NamedResources{}, ctrl.Result{}, err
		}

		if len(cInfos) != 1 {
			return NamedResources{}, ctrl.Result{}, fmt.Errorf("should get info on only one container, got: %d", len(cInfos))
		}

		for _, cInfo := range cInfos {
			if len(cInfo.Stats) < Count {
				r.Log.Info(
					fmt.Sprintf("not enough container stats yet to calculate resources usage, want %d, got %d. Let's wait a bit.",
						Count, len(cInfo.Stats)),
					"container", cStat.Name,
				)
				return NamedResources{},
					ctrl.Result{
						Requeue: true,
						// cInfo.Stats += 1 every 1s on average. We wait the right amount of time +1 second.
						RequeueAfter: time.Duration(Count-len(cInfo.Stats)+1) * time.Second},
					nil
			}

			// get last memory usage
			memoryUsage := cInfo.Stats[len(cInfo.Stats)-1].Memory.Usage
			// calculate cpu usage by comparing extreme timestamps
			cpuUsage := r.calculateCPUusage(cInfo.Stats)

			ress[cStat.Name] = Resources{
				memoryUsage: memoryUsage,
				cpuUsage:    cpuUsage,
			}
		}
	}

	return ress, ctrl.Result{}, nil
}

func (r *Reconciler) calculateCPUusage(cStats []*cadvisorinfo.ContainerStats) float64 {
	cpuStart := cStats[0].Cpu.Usage.Total
	cpuEnd := cStats[len(cStats)-1].Cpu.Usage.Total
	cpuDur := time.Duration(cpuEnd - cpuStart)

	timeStart := cStats[0].Timestamp
	timeEnd := cStats[len(cStats)-1].Timestamp
	timeDur := timeEnd.Sub(timeStart)

	// cpu usage. 0.5 means 0.5 cpu used, 2 means 2 cpu used
	cpuUsage := float64(cpuDur) / float64(timeDur)

	return cpuUsage
}

// throttledTimeStart := cInfo.Stats[0].Cpu.CFS.ThrottledTime
// throttledTimeEnd := cInfo.Stats[len(cInfo.Stats)-1].Cpu.CFS.ThrottledTime
// throttledTimeDur := time.Duration(throttledTimeEnd - throttledTimeStart)
// var cpuThrottled float64
// if cpuDur != 0 {
// 	cpuThrottled = float64(throttledTimeDur) / float64(cpuDur)
// 	// r.Log.Info(fmt.Sprintf("throttled: %d, usage: %d, division: %f", throttledTimeDur, cpuDur, cpuThrottled))
// }
