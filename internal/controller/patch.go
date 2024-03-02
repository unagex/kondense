package controller

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *Reconciler) PatchResources(pod *corev1.Pod, ress Resources) (reconcile.Result, error) {
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
	body := []byte(`{"spec":{"containers":[{"name":"ubuntu", "resources":{"limits":{"memory": "230Mi", "cpu":"100m"},"requests":{"memory": "230Mi", "cpu":"100m"}}}]}}`)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return ctrl.Result{}, err
	}
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/strategic-merge-patch+json")

	resp, err := client.Do(req)
	if err != nil {
		return ctrl.Result{}, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info(string(bodyBytes))
	r.Log.Info("successfuly patched pod with new resources")

	// for _, name := range ress {

	// }
	return reconcile.Result{}, nil
}
