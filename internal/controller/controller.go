package controller

import (
	"context"
	"fmt"
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

	// get cadvisor data
	namedResources, res, err := r.GetCadvisorData(pod)
	if res.Requeue || err != nil {
		return res, err
	}

	// patch pod with cadvisor data
	return r.PatchResources(pod, namedResources)
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
