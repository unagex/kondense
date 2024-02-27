package controller

import (
	"context"
	"fmt"
	"os/exec"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

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
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx).WithName("reconciler")
	r.Log.Info("start reconcile")

	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if k8serrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting pod: %w", err)
	}

	// TODO: check that the pod is in qos guaranteed

	// patch resources without restart to try
	// we cannot use this yet, find another way.
	// pod.Spec.Containers[0].Resources = corev1.ResourceRequirements{
	// 	Limits: corev1.ResourceList{
	// 		corev1.ResourceMemory: resource.MustParse("200Mi"),
	// 		corev1.ResourceCPU:    resource.MustParse("0.3"),
	// 	},
	// 	Requests: corev1.ResourceList{
	// 		corev1.ResourceMemory: resource.MustParse("200Mi"),
	// 		corev1.ResourceCPU:    resource.MustParse("0.2"),
	// 	},
	// }
	// _ = r.Patch(ctx, pod, client.MergeFrom(pod))

	cmd := exec.Command("kubectl", "patch", "pod", pod.Name, "--patch", fmt.Sprintf(`{"spec":{"containers":[{"name": "%s", "resources":{"limits":{"memory": "200Mi", "cpu":"0.2"},"requests":{"memory": "200Mi", "cpu":"0.2"}}}]}}`, pod.Spec.Containers[0].Name))
	output, err := cmd.Output()
	r.Log.Info(string(output))
	if err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("patch done")

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
