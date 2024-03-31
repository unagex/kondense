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
		cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure", "/sys/fs/cgroup/cpu.pressure")
		// we don't need kubectl for kondense container.
		if strings.ToLower(container.Name) == "kondense" {
			cmd = exec.Command("cat", "/sys/fs/cgroup/memory.pressure", "/sys/fs/cgroup/cpu.pressure")
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

	txt := strings.Split(string(output), " ")
	if len(txt) != 17 {
		return fmt.Errorf("error got unexpected pressure stats for container %s: %s", container.Name, txt)
	}

	err = r.UpdateMemStats(container.Name, txt)
	if err != nil {
		return err
	}

	err = r.UpdateCPUStats(container.Name, txt)
	if err != nil {
		return err
	}

	s := r.CStats[container.Name]
	r.L.Printf("container=%s memory_limit=%d memory_time_to_dec=%d memory_total=%d, memory_integral=%d, cpu_limit=%d cpu_time_to_dec=%d cpu_total=%d, cpu_integral=%d",
		container.Name,
		s.Mem.Limit, s.Mem.GraceTicks, s.Mem.PrevTotal, s.Mem.Integral,
		s.Cpu.Limit, s.Cpu.GraceTicks, s.Cpu.PrevTotal, s.Cpu.Integral,
	)

	return nil
}

func (r *Reconciler) UpdateMemStats(containerName string, txt []string) error {
	s := r.CStats[containerName]

	totalTmp := strings.TrimPrefix(txt[4], "total=")
	totalTmp = strings.TrimSuffix(totalTmp, "\nfull")
	total, err := strconv.ParseUint(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	delta := total - s.Mem.PrevTotal
	s.Mem.PrevTotal = total
	s.Mem.Integral += delta

	return nil
}

func (r *Reconciler) UpdateCPUStats(containerName string, txt []string) error {
	s := r.CStats[containerName]

	totalTmp := strings.TrimPrefix(txt[12], "total=")
	totalTmp = strings.TrimSuffix(totalTmp, "\nfull")
	total, err := strconv.ParseUint(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	delta := total - s.Cpu.PrevTotal
	s.Cpu.PrevTotal = total
	s.Cpu.Integral += delta

	return nil
}

func (r *Reconciler) KondenseContainer(container corev1.Container) error {
	memFactor := r.KondenseMemory(container)
	cpuFactor := r.KondenseCPU(container)

	if math.Abs(memFactor) < 0.01 && math.Abs(cpuFactor) < 0.01 {
		return nil
	}

	return r.Adjust(container.Name, memFactor, cpuFactor)
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

func (r *Reconciler) KondenseCPU(container corev1.Container) float64 {
	s := r.CStats[container.Name]

	if s.Cpu.Integral > s.Cpu.TargetPressure {
		// Increase exponentially as we deviate from the target pressure.
		diff := s.Cpu.Integral / s.Cpu.TargetPressure
		adj := math.Pow(float64(diff)/DefaultCPUCoeffInc, 2)
		adj = min(adj*s.Cpu.MaxInc, s.Cpu.MaxInc)

		s.Cpu.GraceTicks = s.Cpu.Interval - 1
		return adj
	}

	// tighten the limit when grace ticks goes to 0.
	if s.Cpu.GraceTicks > 0 {
		s.Cpu.GraceTicks -= 1
		return 0
	}

	// tighten the limit.
	diff := s.Cpu.TargetPressure / max(s.Cpu.Integral, 1)
	adj := math.Pow(float64(diff)/s.Cpu.CoeffDec, 2)
	adj = min(adj*s.Cpu.MaxDec, s.Cpu.MaxDec)

	s.Cpu.GraceTicks = s.Cpu.Interval - 1
	return -adj
}

func (r *Reconciler) Adjust(containerName string, memFactor float64, cpuFactor float64) error {
	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", r.Namespace, r.Name)

	newMemory := uint64(float64(r.CStats[containerName].Mem.Limit) * (1 + memFactor))
	newMemory = min(max(newMemory, r.CStats[containerName].Mem.Min), r.CStats[containerName].Mem.Max)

	newCPU := uint64(float64(r.CStats[containerName].Cpu.Limit) * (1 + cpuFactor))
	newCPU = min(max(newCPU, r.CStats[containerName].Cpu.Min), r.CStats[containerName].Mem.Max)

	body := []byte(fmt.Sprintf(
		`{"spec": {"containers":[{"name":"%s", "resources":{"limits":{"memory": "%d", "cpu": "%dm"},"requests":{"memory": "%d", "cpu": "%dm"}}}]}}`,
		containerName, newMemory, newCPU, newMemory, newCPU))

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

		return r.Adjust(containerName, memFactor, cpuFactor)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch container, want status code: %d, got %d",
			http.StatusOK, resp.StatusCode)
	}
	r.L.Printf("patched container %s with mem factor: %.2f and new memory: %d bytes and with cpu factor : %.2f and new cpu: %d.",
		containerName, memFactor, newMemory, cpuFactor, newCPU)

	r.CStats[containerName].Mem.Integral = 0

	return nil
}
