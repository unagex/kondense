package utils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"
	"slices"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
)

func ContainersToExclude() []string {
	exclude := []string{}
	l, ok := os.LookupEnv("EXCLUDE")
	if ok {
		exclude = strings.Split(l, ",")
	}

	return exclude
}

func GetClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetRawClient() (*http.Client, error) {
	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}, nil
}

func GetBearerToken() (string, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}

	return "Bearer " + string(token), nil
}

func GetMonitorMode(kondenseContainer *corev1.Container, container *corev1.Container) (string) {
	for _, envVar := range container.Env {
		if envVar.Name == strings.ToUpper(container.Name) + "_MODE" {
			return envVar.Value
		}
	}
	return "all";
}