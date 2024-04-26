package main

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/unagex/kondense/pkg/controller"
	"github.com/unagex/kondense/pkg/utils"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// get pod name and namespace.
	name := os.Getenv("HOSTNAME")
	namespaceByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatal().Err(err)
	}
	namespace := string(namespaceByte)

	client, err := utils.GetClient()
	if err != nil {
		log.Fatal().Err(err)
	}
	rawClient, err := utils.GetRawClient()
	if err != nil {
		log.Fatal().Err(err)
	}

	bt, err := utils.GetBearerToken()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get k8s bearer token")
	}

	reconciler := controller.Reconciler{
		Client:    client,
		RawClient: rawClient,

		BearerToken: bt,

		Name:      name,
		Namespace: namespace,
	}

	log.Info().Msg("kondense started")

	reconciler.Reconcile()
}
