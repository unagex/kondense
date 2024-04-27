package controller

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unagex/kondense/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Reconciler struct {
	Client    *kubernetes.Clientset
	RawClient *http.Client

	Mu          sync.Mutex
	BearerToken string

	Namespace string
	Name      string

	CStats ContainerStats
}

func (r *Reconciler) Reconcile() {
	r.CStats = ContainerStats{}

	var start time.Time
	var loopTime time.Duration
	for {
		// one iteration should take 1 second.
		time.Sleep(time.Second - loopTime)
		start = time.Now()

		pod, err := r.Client.CoreV1().Pods(r.Namespace).Get(context.TODO(), r.Name, v1.GetOptions{})
		if err != nil {
			log.Error().Err(err)
			continue
		}
		if pod.Status.QOSClass != corev1.PodQOSGuaranteed {
			log.Error().Msgf("error kondense is only allowed for pods with a QoS class of Guaranteed, got: %s.", pod.Status.QOSClass)
			break
		}

		r.InitCStats(pod)

		var wg sync.WaitGroup
		wg.Add(len(pod.Spec.Containers))

		for _, container := range pod.Spec.Containers {
			go r.ReconcileContainer(pod, container, &wg)
		}

		wg.Wait()

		loopTime = time.Since(start)
	}
}

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
