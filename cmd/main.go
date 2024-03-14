package main

import (
	"context"
	"fmt"
	"log"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"

	"time"
)

func main() {
	// create logger
	l := log.Default()
	l.SetFlags(log.Lshortfile | log.LstdFlags)

	// create k8s client
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		l.Fatal(err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		l.Fatal(err)
	}
	client := mgr.GetClient()

	// start manager
	go func() {
		err = mgr.Start(context.TODO())
		if err != nil {
			l.Fatal(err)
		}
	}()
	time.Sleep(3 * time.Second)

	// get pod name
	podName := os.Getenv("HOSTNAME")

	// get pod namespace
	nsByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		l.Fatal(err)
	}
	ns := string(nsByte)

	// get the containers running alongside the main app
	for {
		time.Sleep(time.Second)

		// get all containers inside current pod
		pod := &corev1.Pod{}
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: podName}, pod)
		if err != nil {
			l.Fatal(err)
		}

		fmt.Println(podName, ns, pod)

		// TODO: make the patch here
		// body := []byte(fmt.Sprintf(
		// 	`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu":"%f"},"requests":{"memory": "%d", "cpu":"%f"}}}]}}`,
		// 	name, newMemory, newCPU, newMemory, newCPU))
	}
}
