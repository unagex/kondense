package controller

import (
	"context"
	"fmt"
	"log"
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
			r.L.Fatal(err)
		}

		fmt.Println(pod)

		// TODO: make the patch here
		// body := []byte(fmt.Sprintf(
		// 	`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu":"%f"},"requests":{"memory": "%d", "cpu":"%f"}}}]}}`,
		// 	name, newMemory, newCPU, newMemory, newCPU))
	}
}
