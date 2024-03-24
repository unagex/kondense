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

	reconciler := controller.Reconciler{
		Client: mgr.GetClient(),
		L:      l,

		Name:      name,
		Namespace: namespace,
	}

	reconciler.Reconcile()
}
