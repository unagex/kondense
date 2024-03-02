package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	cadvisorcli "github.com/google/cadvisor/client/v2"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
)

type Reconciler struct {
	client.Client
	Cclient *cadvisorcli.Client

	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx).WithName("reconciler")

	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if k8serrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting pod: %w", err)
	}

	// check that the pod is in qos guaranteed
	if pod.Status.QOSClass != corev1.PodQOSGuaranteed {
		// check that the condition has not been already added
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "DynamicResizeUnfeasible" {
				return ctrl.Result{}, nil
			}
		}

		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:               "DynamicResizeUnfeasible",
			Status:             "true",
			LastTransitionTime: metav1.Now(),
			Message:            "dynamic resize is only allowed for a pod with a quality of service of guaranteed",
		})
		err = r.Status().Update(ctx, pod)
		if k8serrors.IsConflict(err) {
			// It means the pod has been updated by another controller. Wait 1s before retrying to update.
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating pod status: %w", err)
		}
		r.Log.Info("successfuly updated pod status, dynamic resize is only allowed for pods with a quality of service of guaranteed")

		return ctrl.Result{}, nil
	}

	// Get cAdvisor data ////////////////////////////////////////////////////////////////////////////

	ress, res, err := r.GetCadvisorData(pod)
	if res.Requeue || err != nil {
		return res, err
	}

	_ = ress

	// Patch Pod data ///////////////////////////////////////////////////////////////////////////////

	b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return ctrl.Result{}, err
	}

	var bearer = "Bearer " + string(b)
	url := fmt.Sprintf("https://kubernetes.default.svc.cluster.local/api/v1/namespaces/%s/pods/%s", pod.Namespace, pod.Name)
	body := []byte(`{"spec":{"containers":[{"name":"ubuntu", "resources":{"limits":{"memory": "230Mi", "cpu":"100m"},"requests":{"memory": "230Mi", "cpu":"100m"}}}]}}`)

	patchRequest, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return ctrl.Result{}, err
	}

	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return ctrl.Result{}, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	patchRequest.Header.Add("Authorization", bearer)
	patchRequest.Header.Add("Content-Type", "application/strategic-merge-patch+json")
	resp, err := (&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}).Do(patchRequest)
	if err != nil {
		return ctrl.Result{}, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info(string(bodyBytes))
	r.Log.Info("successfuly patched pod with new resources")

	return ctrl.Result{}, nil
}

func keepCreatePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ce event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	filter := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
		ls := o.GetLabels()
		if ls["app.kubernetes.io/resources-managed-by"] != "kondense" {
			return nil
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: o.GetNamespace(),
					Name:      o.GetName(),
				},
			},
		}
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("kondense").
		Watches(&corev1.Pod{}, filter).
		WithEventFilter(keepCreatePredicate()).
		Complete(r)
}
