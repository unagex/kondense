package controller

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) PatchResources(pod *corev1.Pod, namedResources NamedResources) (reconcile.Result, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return ctrl.Result{}, err
	}
	var bearer = "Bearer " + string(token)

	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return ctrl.Result{}, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", pod.Namespace, pod.Name)

	for name, resources := range namedResources {
		// TODO: update only when memory or cpu changes
		newMemory := resources.memoryUsage * 2
		body := []byte(fmt.Sprintf(
			`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu":"100m"},"requests":{"memory": "%d", "cpu":"100m"}}}]}}`,
			name, newMemory, newMemory))

		req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		if err != nil {
			return ctrl.Result{}, err
		}
		req.Header.Add("Authorization", bearer)
		req.Header.Add("Content-Type", "application/strategic-merge-patch+json")

		// TODO: check that we receive 200 response
		resp, err := client.Do(req)
		if err != nil {
			return ctrl.Result{}, err
		}
		_ = resp

		r.Log.Info(fmt.Sprintf("patched container with memory: %d and cpu: TODO", newMemory))
	}

	return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
}
