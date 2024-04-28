package controller

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
)

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

	if len(s.Cpu.Probes) == int(s.Cpu.Interval) {
		// Pop oldest probe if Probes is full
		s.Cpu.Probes = s.Cpu.Probes[1:]
	}

	p := Probe{
		Total: total,
		T:     s.LastUpdate,
	}
	s.Cpu.Probes = append(s.Cpu.Probes, p)

	// We can calculate when we have 2 or more probes
	if len(s.Cpu.Probes) == 1 {
		return nil
	}

	oldestProbe := s.Cpu.Probes[0]
	newestProbe := s.Cpu.Probes[len(s.Cpu.Probes)-1]

	delta := newestProbe.Total - oldestProbe.Total
	t := newestProbe.T.Sub(oldestProbe.T)

	avgCPU := float64(delta) / max(1, float64(t.Microseconds()))
	avgMCPU := uint64(avgCPU * 1000)
	s.Cpu.Avg = avgMCPU

	return nil
}
