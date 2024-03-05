package controller

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultMemoryRatio = 0.9
	// both in bytes
	// is 10 Mib
	defaultMemoryMin = 10485760
	// is 500 Gib
	defaultMemoryMax = 536870912000

	defaultCPURatio = 0.9
	// both in vCPU
	defaultCPUMin = 0.05
	defaultCPUMax = 100
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
		var memoryRatio float64
		var cpuRatio float64

		memoryRatioString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-memory-ratio", name)]
		memoryRatio, err = strconv.ParseFloat(memoryRatioString, 64)
		if !ok || err != nil || 0 > memoryRatio || memoryRatio > 1 {
			memoryRatio = defaultMemoryRatio
		}

		cpuRatioString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-cpu-ratio", name)]
		cpuRatio, err = strconv.ParseFloat(cpuRatioString, 64)
		if !ok || err != nil || 0 > cpuRatio || cpuRatio > 1 {
			cpuRatio = defaultCPURatio
		}

		// memory is in bytes
		newMemory := int(float64(resources.memoryUsage) * (1 / memoryRatio))
		newMemory = max(defaultMemoryMin, newMemory)
		newMemory = min(defaultMemoryMax, newMemory)

		// cpu is in vCPU
		newCPU := resources.cpuUsage * (1 / cpuRatio)
		newCPU = max(defaultCPUMin, newCPU)
		newCPU = min(defaultCPUMax, newCPU)

		body := []byte(fmt.Sprintf(
			`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu":"%f"},"requests":{"memory": "%d", "cpu":"%f"}}}]}}`,
			name, newMemory, newCPU, newMemory, newCPU))

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

		r.Log.Info(fmt.Sprintf("patched container with memory: %d and cpu: %f", newMemory, newCPU))
	}

	return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
}
