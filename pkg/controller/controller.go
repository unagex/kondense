package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resources struct {
	Memory Pressure
}

func (r *Resources) String() string {
	return fmt.Sprintf("memory: {limit: %d, prevTotal: %d, integral: %d, current: %d}",
		r.Memory.Limit,
		r.Memory.PrevTotal,
		r.Memory.Integral,
		r.Memory.Current)
}

type Pressure struct {
	Limit     int64
	PrevTotal int64
	Integral  int64
	Current   int64

	GraceTicks int
	Interval   int
}

type Reconciler struct {
	client.Client
	L *log.Logger

	Namespace string
	Name      string

	Res map[string]*Resources
}

func (r Reconciler) Reconcile() {
	r.Res = map[string]*Resources{}

	for {
		time.Sleep(1 * time.Second)

		pod := &corev1.Pod{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, pod)
		if err != nil {
			r.L.Println(err)
			continue
		}

		toExclude := []string{}
		l, ok := pod.Annotations["unagex.com/kondense-exclude"]
		if ok {
			toExclude = strings.Split(l, ",")
		}

		// populates memory limit
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if slices.Contains(toExclude, containerStatus.Name) {
				continue
			}

			// initialize container res if not already initialized
			if _, ok := r.Res[containerStatus.Name]; !ok {
				// GraceTicks and Interval default to 6.
				r.Res[containerStatus.Name] = &Resources{
					Memory: Pressure{GraceTicks: 6, Interval: 6}}
			}

			limit := containerStatus.AllocatedResources.Memory().Value()
			r.Res[containerStatus.Name].Memory.Limit = limit
		}

		for _, container := range pod.Spec.Containers {
			if slices.Contains(toExclude, container.Name) {
				continue
			}

			// initialize container res if not already initialized
			if _, ok := r.Res[container.Name]; !ok {
				r.Res[container.Name] = &Resources{}
			}

			// 1. get pressures with kubectl for every containers.
			//
			// cat need to be installed in the kondensed container
			// kubectl exec -i test-kondense-7c8f646f79-5l824 -c ubuntu -- cat /sys/fs/cgroup/cpu.pressure
			cmd := exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/cpu.pressure")
			cpuPressureOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}
			_ = cpuPressureOutput

			cmd = exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.pressure")
			memoryPressureOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}

			// initialize memory to the current use.
			cmd = exec.Command("kubectl", "exec", "-i", r.Name, "-c", container.Name, "--", "cat", "/sys/fs/cgroup/memory.current")
			memoryCurrentOutput, err := cmd.Output()
			if err != nil {
				r.L.Println(err)
				continue
			}
			_ = memoryCurrentOutput

			memoryPressureTmp := strings.Split(string(memoryPressureOutput), " ")[4]
			memoryPressureTmp = strings.TrimPrefix(memoryPressureTmp, "total=")
			memoryPressureTmp = strings.TrimSuffix(memoryPressureTmp, "\nfull")
			memoryPressure, err := strconv.ParseInt(memoryPressureTmp, 10, 64)
			if err != nil {
				r.L.Println(err)
				continue
			}

			delta := memoryPressure - r.Res[container.Name].Memory.PrevTotal
			r.Res[container.Name].Memory.PrevTotal = memoryPressure
			r.Res[container.Name].Memory.Integral += delta

			// conf.pressure = 10 * 1000 as default
			if r.Res[container.Name].Memory.Integral > 10*1000 {
				// Back off exponentially as we deviate from the target pressure.
				diff := r.Res[container.Name].Memory.Integral / (10 * 1000)
				// coeff_backoff = 20 as default
				adj := math.Pow(float64(diff/20), 2)
				// max_backoff = 1 as default
				adj = min(adj*1, 1)

				err = r.Adjust(container.Name, adj)
				if err != nil {
					r.L.Println(err)
				}
				r.Res[container.Name].Memory.GraceTicks = r.Res[container.Name].Memory.Interval - 1
				continue
			}

			if r.Res[container.Name].Memory.GraceTicks > 0 {
				r.Res[container.Name].Memory.GraceTicks -= 1
				continue
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
	}
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
