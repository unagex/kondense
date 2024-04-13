package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"strconv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

type MemPsi struct {
	Some MemMetrics
	Full MemMetrics
}

// MemMetrics holds the actual metric data.
type MemMetrics struct {
	Avg10  float64
	Avg60  float64
	Avg300 float64
	Total  uint64
}

func parseMemPsiOutput(output string) (*MemPsi, error) {
	// Regex to capture "some" and "full" separately
	re := regexp.MustCompile(`(some|full) avg10=(\d+\.\d+) avg60=(\d+\.\d+) avg300=(\d+\.\d+) total=(\d+)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if matches == nil || len(matches) < 2 {
			return nil, fmt.Errorf("expected data for both 'some' and 'full' not found")
	}

	psi := &MemPsi{}
	for _, match := range matches {
			metrics := MemMetrics{
					Avg10:  parseFloat(match[2]),
					Avg60:  parseFloat(match[3]),
					Avg300: parseFloat(match[4]),
					Total:  parseUint(match[5]),
			}

			if match[1] == "some" {
					psi.Some = metrics
			} else if match[1] == "full" {
					psi.Full = metrics
			}
	}

	return psi, nil
}

func parseFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func parseUint(value string) uint64 {
	result, _ := strconv.ParseUint(value, 10, 64)
	return result
}
