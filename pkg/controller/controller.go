package controller

import (
	"context"
	"log"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client.Client
	L *log.Logger

	Namespace string
	Name      string
}

func (r Reconciler) Reconcile() {
	for {
		time.Sleep(5 * time.Second)

		// get all containers inside current pod
		pod := &corev1.Pod{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, pod)
		if err != nil {
			r.L.Println(err)
			continue
		}

		for _, container := range pod.Spec.Containers {
			_ = container
			// 1. get pressures with kubectl for every containers.
			//
			// cat need to be installed in the kondensed container
			// kubectl exec -i test-kondense-7c8f646f79-5l824 -c ubuntu -- cat /proc/pressure/cpu > ubuntu-cpu
			cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/proc/pressure/cpu")
			// r.L.Println(strings.Join(cmd.Args, " "))
			cpuOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}

			cmd = exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/proc/pressure/memory")
			memoryOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}

			r.L.Println(string(cpuOutput), string(memoryOutput))
			//

			// 2. patch container resource for every containers.
		}
	}
}
