package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	//"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayClassReconciler struct {
	client        client.Client
	dynamicClient dynamic.Interface
	Scheme        *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses/finalizers,verbs=update

func NewGatewayClassController(mgr ctrl.Manager) *GatewayClassReconciler {
	r := &GatewayClassReconciler{
		client:        mgr.GetClient(),
		dynamicClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		Scheme:        mgr.GetScheme(),
	}
	return r
}

func (r *GatewayClassReconciler) Client() client.Client {
	return r.client
}

func (r *GatewayClassReconciler) DynamicClient() dynamic.Interface {
	return r.dynamicClient
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	gwc := &gateway.GatewayClass{}
	err := r.client.Get(ctx, req.NamespacedName, gwc)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconcile", "gatewayclass", gwc)

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

func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.GatewayClass{}).
		Complete(r)
}
