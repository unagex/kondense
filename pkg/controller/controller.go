package controller

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
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
		time.Sleep(loopTime - time.Second)
		start = time.Now()

		pod, err := r.Client.CoreV1().Pods(r.Namespace).Get(context.TODO(), r.Name, v1.GetOptions{})
		if err != nil {
			log.Error().Err(err)
			continue
		}
		if pod.Status.QOSClass != corev1.PodQOSGuaranteed {
			log.Error().Msgf("error kondense is only allowed for pods with a QoS class of Guaranteed, got: %s.", pod.Status.QOSClass)
			continue
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
