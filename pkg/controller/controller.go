package controller

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Reconciler struct {
	Client    *kubernetes.Clientset
	RawClient *http.Client
	L         *log.Logger

	Mu          sync.Mutex
	BearerToken string

	Namespace string
	Name      string

	CStats ContainerStats
}

func (r *Reconciler) Reconcile() {
	r.CStats = ContainerStats{}

	for {
		time.Sleep(1 * time.Second)

		pod, err := r.Client.CoreV1().Pods(r.Namespace).Get(context.TODO(), r.Name, v1.GetOptions{})
		//  Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, pod)
		if err != nil {
			r.L.Println(err)
			continue
		}
		if pod.Status.QOSClass != corev1.PodQOSGuaranteed {
			r.L.Printf("error kondense is only allowed for pods with a QoS class of Guaranteed, got: %s.", pod.Status.QOSClass)
			continue
		}

		r.InitCStats(pod)

		var wg sync.WaitGroup
		wg.Add(len(pod.Spec.Containers))

		for _, container := range pod.Spec.Containers {
			go r.ReconcileContainer(pod, container, &wg)
		}

		wg.Wait()
	}
}
