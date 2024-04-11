package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/unagex/kondense/pkg/controller"
	"github.com/unagex/kondense/pkg/utils"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.TraceLevel).With().Timestamp().Logger()

	// get pod name and namespace.
	name := os.Getenv("HOSTNAME")
	namespaceByte, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		l.Fatal().Err(err)
	}
	namespace := string(namespaceByte)

	client, err := utils.GetClient()
	if err != nil {
		l.Fatal().Err(err)
	}
	rawClient, err := utils.GetRawClient()
	if err != nil {
		l.Fatal().Err(err)
	}

	bt, err := utils.GetBearerToken()
	if err != nil {
		l.Fatal().Err(err).Msg("failed to get k8s bearer token")
	}

	reconciler := controller.Reconciler{
		Client:    client,
		RawClient: rawClient,
		L:         &l,

		BearerToken: bt,

		Name:      name,
		Namespace: namespace,
	}

	reconciler.Reconcile()
}
