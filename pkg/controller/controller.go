package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client.Client
	L *log.Logger
}

func (r Reconciler) Reconcile() {
	// get pod name
	podName := os.Getenv("HOSTNAME")

	// get pod namespace
	nsByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		r.L.Fatal(err)
	}
	ns := string(nsByte)

	// get the containers running alongside the main app
	for {
		time.Sleep(time.Second)

		// get all containers inside current pod
		pod := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: podName}, pod)
		if err != nil {
			r.L.Fatal(err)
		}

		fmt.Println(podName, ns, pod)

		// TODO: make the patch here
		// body := []byte(fmt.Sprintf(
		// 	`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu":"%f"},"requests":{"memory": "%d", "cpu":"%f"}}}]}}`,
		// 	name, newMemory, newCPU, newMemory, newCPU))
	}
}
