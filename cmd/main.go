package main

import (
	"context"
	"log"

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

	reconciler := controller.Reconciler{
		Client: mgr.GetClient(),
		L:      l,
	}

	reconciler.Reconcile()
}
