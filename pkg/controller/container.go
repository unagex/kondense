package controller

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

func (r Reconciler) KondenseContainer(pod *corev1.Pod, container corev1.Container, wg *sync.WaitGroup) {
	defer wg.Done()

	exclude := containersToExclude(pod)
	if slices.Contains(exclude, container.Name) {
		return
	}

	err := r.UpdateStats(pod, container)
	if err != nil {
		r.L.Print(err)
		return
	}

	// conf.pressure = 10 * 1000 as default
	if r.Res[container.Name].Memory.Integral > 10*1000 {
		// Back off exponentially as we deviate from the target pressure.
		diff := r.Res[container.Name].Memory.Integral / (10 * 1000)
		// coeff_backoff = 20 as default
		adj := math.Pow(float64(diff/20), 2)
		// max_backoff = 1 as default
		adj = min(adj*1, 1)

		err := r.Adjust(container.Name, adj)
		if err != nil {
			r.L.Println(err)
		}
		r.Res[container.Name].Memory.GraceTicks = r.Res[container.Name].Memory.Interval - 1
		return
	}

	if r.Res[container.Name].Memory.GraceTicks > 0 {
		r.Res[container.Name].Memory.GraceTicks -= 1
		return
	}
	// Tighten the limit.
	diff := (10 * 1000) / max(r.Res[container.Name].Memory.Integral, 1)
	// coeffProbe default to 10
	adj := math.Pow(float64(diff/10), 2)
	// max_probe default is 0.01
	adj = min(adj*0.01, 0.01)

	err = r.Adjust(container.Name, -adj)
	if err != nil {
		r.L.Println(err)
	}
	r.Res[container.Name].Memory.GraceTicks = r.Res[container.Name].Memory.Interval - 1

}

func (r Reconciler) UpdateStats(pod *corev1.Pod, container corev1.Container) error {
	cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	totalTmp := strings.Split(string(output), " ")[4]
	totalTmp = strings.TrimPrefix(totalTmp, "total=")
	totalTmp = strings.TrimSuffix(totalTmp, "\nfull")
	total, err := strconv.ParseInt(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	delta := total - r.Res[container.Name].Memory.PrevTotal
	r.Res[container.Name].Memory.PrevTotal = total
	r.Res[container.Name].Memory.Integral += delta

	avg10Tmp := strings.Split(string(output), " ")[1]
	avg10Tmp = strings.TrimPrefix(avg10Tmp, "avg10=")
	avg10, err := strconv.ParseFloat(avg10Tmp, 64)
	if err != nil {
		return err
	}
	r.Res[container.Name].Memory.AVG10 = avg10

	avg60Tmp := strings.Split(string(output), " ")[2]
	avg60Tmp = strings.TrimPrefix(avg60Tmp, "avg60=")
	avg60, err := strconv.ParseFloat(avg60Tmp, 64)
	if err != nil {
		return err
	}
	r.Res[container.Name].Memory.AVG60 = avg60

	avg300Tmp := strings.Split(string(output), " ")[3]
	avg300Tmp = strings.TrimPrefix(avg300Tmp, "avg300=")
	avg300, err := strconv.ParseFloat(avg300Tmp, 64)
	if err != nil {
		return err
	}
	r.Res[container.Name].Memory.AVG300 = avg300

	r.L.Printf("container=%s limit=%d memory_pressure_avg10=%f memory_pressure_avg60=%f memory_pressure_avg300=%f time_to_probe=%d total=%d delta=%d integral=%d",
		container.Name, r.Res[container.Name].Memory.Limit, avg10, avg60, avg300,
		r.Res[container.Name].Memory.GraceTicks, total, delta, r.Res[container.Name].Memory.Integral)

	return nil
}

func (r Reconciler) Adjust(containerName string, factor float64) error {
	client, err := getK8SClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", r.Namespace, r.Name)

	newMemory := int(float64(r.Res[containerName].Memory.Limit) * (1 + factor))
	body := []byte(fmt.Sprintf(
		`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d"},"requests":{"memory": "%d"}}}]}}`,
		containerName, newMemory, newMemory))

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return err
	}
	var bearer = "Bearer " + string(token)
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/strategic-merge-patch+json")

	// TODO: check that we receive 200 response
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch container, want status code: %d, got %d",
			http.StatusOK, resp.StatusCode)
	}
	r.L.Printf("patched container %s with factor: %f and new memory: %d", containerName, factor, newMemory)

	r.Res[containerName].Memory.Integral = 0

	return nil
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
