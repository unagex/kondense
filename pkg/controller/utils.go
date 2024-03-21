package controller

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func containersToExclude(pod *corev1.Pod) []string {
	exclude := []string{}
	l, ok := pod.Annotations["unagex.com/kondense-exclude"]
	if ok {
		exclude = strings.Split(l, ",")
	}

	return exclude
}

func getK8SClient() (*http.Client, error) {
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
