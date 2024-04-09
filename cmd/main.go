package main

import (
	"log"
	"os"

	"github.com/unagex/kondense/pkg/controller"
	"github.com/unagex/kondense/pkg/utils"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// create logger.
	l := log.Default()
	l.SetFlags(log.Lshortfile | log.LstdFlags)

	// get pod name and namespace.
	name := os.Getenv("HOSTNAME")
	namespaceByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		l.Fatal(err)
	}
	namespace := string(namespaceByte)

	client, err := utils.GetClient()
	if err != nil {
		l.Fatal(err)
	}
	rawClient, err := utils.GetRawClient()
	if err != nil {
		l.Fatal(err)
	}

	bt, err := utils.GetBearerToken()
	if err != nil {
		l.Fatalf("failed to get k8s bearer token: %s", err)
	}

	reconciler := controller.Reconciler{
		Client:    client,
		RawClient: rawClient,
		L:         l,

		BearerToken: bt,

		Name:      name,
		Namespace: namespace,
	}

	reconciler.Reconcile()
}
