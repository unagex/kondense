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
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultMemoryRatio = 0.9
	// both in bytes
	// is 10 Mi
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

		memoryMinString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-memory-min", name)]
		memoryMinResource, err := resource.ParseQuantity(memoryMinString)
		memoryMin := memoryMinResource.Value()
		if !ok || err != nil || 0 > memoryMin {
			memoryMin = defaultMemoryMin
		}

		memoryMaxString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-memory-max", name)]
		memoryMaxResource, err := resource.ParseQuantity(memoryMaxString)
		memoryMax := memoryMaxResource.Value()
		if !ok || err != nil || memoryMax < memoryMin {
			memoryMax = defaultMemoryMax
		}

		cpuMinString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-cpu-min", name)]
		cpuMinResource, err := resource.ParseQuantity(cpuMinString)
		cpuMin := cpuMinResource.AsApproximateFloat64()
		if !ok || err != nil || 0 > cpuMin {
			cpuMin = defaultCPUMin
		}

		cpuMaxString, ok := pod.Annotations[fmt.Sprintf("unagex.com/%s-cpu-max", name)]
		cpuMaxResource, err := resource.ParseQuantity(cpuMaxString)
		cpuMax := cpuMaxResource.AsApproximateFloat64()
		if !ok || err != nil || cpuMax < cpuMin {
			cpuMax = defaultCPUMax
		}

		// memory is in bytes
		newMemory := int64(float64(resources.memoryUsage) * (1 / memoryRatio))
		newMemory = max(memoryMin, newMemory)
		newMemory = min(memoryMax, newMemory)

		// cpu is in vCPU
		newCPU := resources.cpuUsage * (1 / cpuRatio)
		newCPU = max(cpuMin, newCPU)
		newCPU = min(cpuMax, newCPU)

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
