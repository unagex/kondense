package controller

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unagex/kondense/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) ReconcileContainer(pod *corev1.Pod, container corev1.Container, wg *sync.WaitGroup) {
	defer wg.Done()

	exclude := utils.ContainersToExclude()
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

func (r *Reconciler) UpdateStats(pod *corev1.Pod, container corev1.Container) error {
	var err error
	var output []byte
	for i := 0; i < 3; i++ {
		cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure")
		// we don't need kubectl for kondense container.
		if strings.ToLower(container.Name) == "kondense" {
			cmd = exec.Command("cat", "/sys/fs/cgroup/memory.pressure")
		}
		output, err = cmd.Output()
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return err
	}

	some := strings.Split(string(output), " ")
	if len(some) != 9 {
		return fmt.Errorf("error got unexpected memory.pressure for container %s: %s", container.Name, output)
	}

	totalTmp := strings.TrimPrefix(some[4], "total=")
	totalTmp = strings.TrimSuffix(totalTmp, "\nfull")
	total, err := strconv.ParseUint(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	s := r.CStats[container.Name]

	delta := total - s.Mem.PrevTotal
	s.Mem.PrevTotal = total
	s.Mem.Integral += delta

	avg10Tmp := strings.TrimPrefix(some[1], "avg10=")
	avg10, err := strconv.ParseFloat(avg10Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG10 = avg10

	avg60Tmp := strings.TrimPrefix(some[2], "avg60=")
	avg60, err := strconv.ParseFloat(avg60Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG60 = avg60

	avg300Tmp := strings.TrimPrefix(some[3], "avg300=")
	avg300, err := strconv.ParseFloat(avg300Tmp, 64)
	if err != nil {
		return err
	}
	s.Mem.AVG300 = avg300

	r.L.Printf("container=%s limit=%d memory_pressure_avg10=%.2f memory_pressure_avg60=%.2f memory_pressure_avg300=%.2f time_to_dec=%d total=%d delta=%d integral=%d",
		container.Name, s.Mem.Limit,
		avg10, avg60, avg300,
		s.Mem.GraceTicks, total, delta, s.Mem.Integral)

	return nil
}

func (r *Reconciler) KondenseContainer(container corev1.Container) error {
	MemFactor := r.KondenseMemory(container)
	// CpuFactor := r.KondenseCPU(container)

	return r.Adjust(container.Name, MemFactor)
}

func (r *Reconciler) KondenseMemory(container corev1.Container) float64 {
	s := r.CStats[container.Name]

	if s.Mem.Integral > s.Mem.TargetPressure {
		// Increase exponentially as we deviate from the target pressure.
		diff := s.Mem.Integral / s.Mem.TargetPressure
		adj := math.Pow(float64(diff)/DefaultMemCoeffInc, 2)
		adj = min(adj*s.Mem.MaxInc, s.Mem.MaxInc)

		s.Mem.GraceTicks = s.Mem.Interval - 1
		// return r.Adjust(container.Name, adj)
		return adj
	}

	// tighten the limit when grace ticks goes to 0.
	if s.Mem.GraceTicks > 0 {
		s.Mem.GraceTicks -= 1
		return 0
		// return nil
	}

	// tighten the limit.
	diff := s.Mem.TargetPressure / max(s.Mem.Integral, 1)
	adj := math.Pow(float64(diff)/s.Mem.CoeffDec, 2)
	adj = min(adj*s.Mem.MaxDec, s.Mem.MaxDec)

	s.Mem.GraceTicks = s.Mem.Interval - 1

	// return r.Adjust(container.Name, -adj)
	return -adj
}

func (r *Reconciler) Adjust(containerName string, factor float64) error {
	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", r.Namespace, r.Name)

	newMemory := uint64(float64(r.CStats[containerName].Mem.Limit) * (1 + factor))
	newMemory = min(max(newMemory, r.CStats[containerName].Mem.Min), r.CStats[containerName].Mem.Max)

	body := []byte(fmt.Sprintf(
		`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d"},"requests":{"memory": "%d"}}}]}}`,
		containerName, newMemory, newMemory))

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	r.Mu.Lock()
	bt := r.BearerToken
	r.Mu.Unlock()

	req.Header.Add("Authorization", bt)
	req.Header.Add("Content-Type", "application/strategic-merge-patch+json")

	resp, err := r.RawClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// renew k8s token
		bt, err := utils.GetBearerToken()
		if err != nil {
			r.L.Fatalf("failed to renew k8s bearer token: %s", err)
		}

		r.Mu.Lock()
		r.L.Print("renewed k8s bearer token.")
		r.BearerToken = bt
		r.Mu.Unlock()

		return r.Adjust(containerName, factor)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch container, want status code: %d, got %d",
			http.StatusOK, resp.StatusCode)
	}
	r.L.Printf("patched container %s with factor: %.2f and new memory: %d bytes.", containerName, factor, newMemory)

	r.CStats[containerName].Mem.Integral = 0

	return nil
}
