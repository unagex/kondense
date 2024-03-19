package controller

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resources struct {
	Memory Pressure
}

func (r *Resources) String() string {
	return fmt.Sprintf("memory: {total: %d, prevTotal: %d, limit: %d, current: %d}",
		r.Memory.Total,
		r.Memory.PrevTotal,
		r.Memory.Limit,
		r.Memory.Current)
}

type Pressure struct {
	Total     int
	PrevTotal int
	Limit     int64
	Current   int
}

type Reconciler struct {
	client.Client
	L *log.Logger

	Namespace string
	Name      string
}

func (r Reconciler) Reconcile() {
	res := map[string]*Resources{}

	for {
		time.Sleep(5 * time.Second)

		// get all containers inside current pod
		pod := &corev1.Pod{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, pod)
		if err != nil {
			r.L.Println(err)
			continue
		}

		// populates memory limit
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// initialize container res if not already initialized
			if _, ok := res[containerStatus.Name]; !ok {
				res[containerStatus.Name] = &Resources{}
			}

			limit := containerStatus.AllocatedResources.Memory().Value()
			res[containerStatus.Name].Memory.Limit = limit
		}

		for _, container := range pod.Spec.Containers {
			// initialize container res if not already initialized
			if _, ok := res[container.Name]; !ok {
				res[container.Name] = &Resources{}
			}

			// 1. get pressures with kubectl for every containers.
			//
			// cat need to be installed in the kondensed container
			// kubectl exec -i test-kondense-7c8f646f79-5l824 -c ubuntu -- cat /sys/fs/cgroup/cpu.pressure
			cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/cpu.pressure")
			cpuPressureOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}
			_ = cpuPressureOutput

			cmd = exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure")
			memoryPressureOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}

			// initialize memory to the current use.
			cmd = exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.current")
			memoryCurrentOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}
			_ = memoryCurrentOutput

			memoryPressureTmp := strings.Split(string(memoryPressureOutput), " ")[4]
			memoryPressureTmp = strings.TrimPrefix(memoryPressureTmp, "total=")
			memoryPressureTmp = strings.TrimSuffix(memoryPressureTmp, "\nfull")
			memoryPressure, err := strconv.Atoi(memoryPressureTmp)
			if err != nil {
				r.L.Println(err)
				continue
			}

			// update variables
			res[container.Name].Memory.PrevTotal = res[container.Name].Memory.Total
			res[container.Name].Memory.Total = memoryPressure

			r.L.Println(res)

			// 2. patch container resource for every containers.
		}
	}
}
