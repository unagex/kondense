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

	"github.com/rs/zerolog/log"
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
		log.Error().Err(err)
		return
	}

	err = r.KondenseContainer(container)
	if err != nil {
		log.Error().Err(err)
	}
}

func (r *Reconciler) UpdateStats(pod *corev1.Pod, container corev1.Container) error {
	var err error
	var output []byte
	for i := 0; i < 3; i++ {
		cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure", "/sys/fs/cgroup/cpu.stat")
		// we don't need kubectl for kondense container.
		if strings.ToLower(container.Name) == "kondense" {
			cmd = exec.Command("cat", "/sys/fs/cgroup/memory.pressure", "/sys/fs/cgroup/cpu.stat")
		}
		output, err = cmd.Output()
		if err == nil {
			r.CStats[container.Name].LastUpdate = time.Now()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return err
	}

	txt := strings.Split(string(output), " ")
	if len(txt) != 15 {
		return fmt.Errorf("error got unexpected stats for container %s: %s", container.Name, txt)
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
	log.Info().
		Str("container", container.Name).
		Int64("memory_limit", s.Mem.Limit).
		Uint64("memory_time to decrease", s.Mem.GraceTicks).
		Uint64("memory_total", s.Mem.PrevTotal).
		Uint64("integral", s.Mem.Integral).
		Int64("cpu_limit", s.Cpu.Limit).
		Uint64("cpu_average", s.Cpu.Avg).
		Msg("updated stats")

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

	totalTmp := strings.TrimSuffix(txt[9], "\nuser_usec")
	total, err := strconv.ParseUint(totalTmp, 10, 64)
	if err != nil {
		return err
	}

	if len(s.Cpu.Usage) == int(s.Cpu.Interval) {
		// Pop oldest probe if Usage is full
		s.Cpu.Usage = s.Cpu.Usage[1:]
	}

	p := CPUProbe{
		Usage: total,
		T:     s.LastUpdate,
	}
	s.Cpu.Usage = append(s.Cpu.Usage, p)

	// We can calculate when we have 2 or more probes
	if len(s.Cpu.Usage) == 1 {
		return nil
	}

	oldestProbe := s.Cpu.Usage[0]
	newestProbe := s.Cpu.Usage[len(s.Cpu.Usage)-1]

	delta := newestProbe.Usage - oldestProbe.Usage
	t := newestProbe.T.Sub(oldestProbe.T)

	avgCPU := float64(delta) / max(1, float64(t.Microseconds()))
	avgMCPU := uint64(avgCPU * 1000)
	s.Cpu.Avg = avgMCPU

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
		diff := s.Mem.Integral / max(1, s.Mem.TargetPressure)
		adj := math.Pow(float64(diff)/DefaultMemCoeffInc, 2)
		adj = min(adj*s.Mem.MaxInc, s.Mem.MaxInc)

		s.Mem.GraceTicks = s.Mem.Interval - 1
		return adj
	}

	// tighten the limit when grace ticks goes to 0.
	if s.Mem.GraceTicks > 0 {
		s.Mem.GraceTicks -= 1
		return 0
	}

	// tighten the limit.
	diff := s.Mem.TargetPressure / max(s.Mem.Integral, 1)
	adj := math.Pow(float64(diff)/s.Mem.CoeffDec, 2)
	adj = min(adj*s.Mem.MaxDec, s.Mem.MaxDec)

	s.Mem.GraceTicks = s.Mem.Interval - 1
	return -adj
}

func (r *Reconciler) KondenseCPU(container corev1.Container) float64 {
	s := r.CStats[container.Name]

	newLimit := float64(s.Cpu.Avg) / max(0.1, s.Cpu.TargetAvg)
	adj := newLimit/float64(s.Cpu.Limit) - 1

	if adj > 0 {
		adj = adj + math.Pow(float64(s.Cpu.Coeff)*adj, 2)
		return min(adj, s.Cpu.MaxInc)
	}

	return max(adj, -s.Cpu.MaxDec)
}

func (r *Reconciler) Adjust(containerName string, memFactor, cpuFactor float64) error {
	s := r.CStats[containerName]
	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", r.Namespace, r.Name)

	newMemory := uint64(float64(s.Mem.Limit) * (1 + memFactor))
	newMemory = min(max(newMemory, s.Mem.Min), s.Mem.Max)

	newCPU := uint64(float64(s.Cpu.Limit) * (1 + cpuFactor))
	newCPU = min(max(newCPU, s.Cpu.Min), s.Cpu.Max)

	MemUpdate := newMemory != uint64(s.Mem.Limit)
	CPUUpdate := newCPU != uint64(s.Cpu.Limit)
	if !MemUpdate && !CPUUpdate {
		return nil
	}

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
			log.Fatal().Msgf("failed to renew k8s bearer token: %s", err)
		}

		r.Mu.Lock()
		log.Info().Msg("renewed k8s bearer token.")
		r.BearerToken = bt
		r.Mu.Unlock()

		return r.Adjust(containerName, memFactor, cpuFactor)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch container, want status code: %d, got %d",
			http.StatusOK, resp.StatusCode)
	}

	memFactorLog, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", memFactor), 64)
	cpuFactorLog, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", cpuFactor), 64)
	log.Info().
		Str("container", containerName).
		Float64("memory_factor", memFactorLog).
		Uint64("new_memory", newMemory).
		Float64("cpu_factor", cpuFactorLog).
		Uint64("new_cpu", newCPU).
		Msg("patched container")

	s.Mem.Integral = 0

	return nil
}
