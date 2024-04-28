package controller

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/unagex/kondense/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

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
