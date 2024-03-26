package main

import (
	"context"
	"log"
	"os"

	"github.com/unagex/kondense/pkg/controller"
	"github.com/unagex/kondense/pkg/manager"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// create logger
	l := log.Default()
	l.SetFlags(log.Lshortfile | log.LstdFlags)

	// create manager
	mgr, err := manager.Create()
	if err != nil {
		l.Fatal(err)
	}

	// start manager
	go func() {
		err = mgr.Start(context.TODO())
		if err != nil {
			l.Fatal(err)
		}
	}()

	// get pod name and namespace
	name := os.Getenv("HOSTNAME")
	namespaceByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		l.Fatal(err)
	}
	namespace := string(namespaceByte)

	K8sClient, err := controller.GetK8SClient()
	if err != nil {
		l.Fatal(err)
	}

	bt, err := controller.GetBearerToken()
	if err != nil {
		l.Fatalf("failed to get k8s bearer token: %s", err)
	}

	reconciler := controller.Reconciler{
		Client:    mgr.GetClient(),
		K8sClient: K8sClient,
		L:         l,

		BearerToken: bt,

		Name:      name,
		Namespace: namespace,
	}

	reconciler.Reconcile()
}
