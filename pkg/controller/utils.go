package controller

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"
)

func containersToExclude() []string {
	exclude := []string{}
	l, ok := os.LookupEnv("EXCLUDE")
	if ok {
		exclude = strings.Split(l, ",")
	}

	return exclude
}

func GetK8SClient() (*http.Client, error) {
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
