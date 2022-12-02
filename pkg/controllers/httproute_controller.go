package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	//"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	lister "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
)

type HTTPRouteReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	gatewayLister lister.GatewayLister
	//gatewayClassLister lister.GatewayClassLister
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/finalizers,verbs=update

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	rt := &gateway.HTTPRoute{}
	err := r.Get(ctx, req.NamespacedName, rt)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconcile", "httproute", rt)

	// gw, err := r.gatewayLister.Gateways(req.Namespace).Get(req.Name)
	// if err != nil || gw == nil {
	// 	// we'll ignore not-found errors, since they can't be fixed by an immediate
	// 	// requeue (we'll need to wait for a new notification), and we can get them
	// 	// on deleted requests.
	// 	if err := controllers.IgnoreNotFound(err); err != nil {
	// 		log.Errorf("unable to fetch Gateway: %v", err)
	// 		return err
	// 	}
	// 	return nil
	// }

	return ctrl.Result{}, nil
}

func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.HTTPRoute{}).
		Complete(r)
}
