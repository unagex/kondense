package controller

import (
	"bytes"
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

func (r Reconciler) ReconcileContainer(pod *corev1.Pod, container corev1.Container, wg *sync.WaitGroup) {
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

	err = r.KondenseContainer(container)
	if err != nil {
		r.L.Print(err)
	}
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
	total, err := strconv.ParseUint(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	s := r.CStats[container.Name]

	delta := total - s.Mem.PrevTotal
	s.Mem.PrevTotal = total
	s.Mem.Integral += delta

	avg10Tmp := strings.Split(string(output), " ")[1]
	avg10Tmp = strings.TrimPrefix(avg10Tmp, "avg10=")
	avg10, err := strconv.ParseFloat(avg10Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG10 = avg10

	avg60Tmp := strings.Split(string(output), " ")[2]
	avg60Tmp = strings.TrimPrefix(avg60Tmp, "avg60=")
	avg60, err := strconv.ParseFloat(avg60Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG60 = avg60

	avg300Tmp := strings.Split(string(output), " ")[3]
	avg300Tmp = strings.TrimPrefix(avg300Tmp, "avg300=")
	avg300, err := strconv.ParseFloat(avg300Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG300 = avg300

	r.L.Printf("container=%s limit=%d memory_pressure_avg10=%.2f memory_pressure_avg60=%.2f memory_pressure_avg300=%.2f time_to_probe=%d total=%d delta=%d integral=%d",
		container.Name, s.Mem.Limit,
		avg10, avg60, avg300,
		s.Mem.GraceTicks, total, delta, s.Mem.Integral)

	return nil
}

func (r Reconciler) KondenseContainer(container corev1.Container) error {
	s := r.CStats[container.Name]

	if s.Mem.Integral > s.Mem.TargetPressure {
		// Back off exponentially as we deviate from the target pressure.
		diff := s.Mem.Integral / s.Mem.TargetPressure
		// coeff_backoff = 20 as default
		adj := math.Pow(float64(diff/20), 2)
		// max_backoff = 1 as default
		adj = min(adj*1, 1)

		s.Mem.GraceTicks = s.Mem.Interval - 1
		return r.Adjust(container.Name, adj)
	}

	// tighten the limit when grace ticks goes to 0.
	if s.Mem.GraceTicks > 0 {
		s.Mem.GraceTicks -= 1
		return nil
	}

	// Tighten the limit.
	diff := s.Mem.TargetPressure / max(s.Mem.Integral, 1)
	// coeffProbe default to 10
	adj := math.Pow(float64(diff/10), 2)
	adj = min(adj*s.Mem.MaxProbe, s.Mem.MaxProbe)

	s.Mem.GraceTicks = s.Mem.Interval - 1

	return r.Adjust(container.Name, -adj)
}

func (r Reconciler) Adjust(containerName string, factor float64) error {
	client, err := getK8SClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", r.Namespace, r.Name)

	newMemory := int(float64(r.CStats[containerName].Mem.Limit) * (1 + factor))
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

	r.CStats[containerName].Mem.Integral = 0

	return nil
}
