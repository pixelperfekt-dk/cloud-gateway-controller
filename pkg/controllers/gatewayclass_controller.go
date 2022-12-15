package controllers

import (
	"context"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayClassReconciler struct {
	client        client.Client
	dynamicClient dynamic.Interface
	scheme        *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=gatewayclasses/finalizers,verbs=update

func NewGatewayClassController(mgr ctrl.Manager) *GatewayClassReconciler {
	r := &GatewayClassReconciler{
		client:        mgr.GetClient(),
		dynamicClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		scheme:        mgr.GetScheme(),
	}
	return r
}

func (r *GatewayClassReconciler) Client() client.Client {
	return r.client
}

func (r *GatewayClassReconciler) DynamicClient() dynamic.Interface {
	return r.dynamicClient
}

func (r *GatewayClassReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//log := log.FromContext(ctx)

	// gwc := &gateway.GatewayClass{}
	// err := r.client.Get(ctx, req.NamespacedName, gwc)
	// if err != nil {
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }
	// log.Info("reconcile", "gatewayclass", gwc)

	gwc, configmap, err := lookupGatewayClass(r, ctx, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if gwc == nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if configmap == nil {
	    meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
		    Type:   string(gateway.GatewayClassConditionStatusAccepted),
		    Status: "False",
		    Reason: string(gateway.GatewayClassReasonInvalidParameters)})
	} else {
	    meta.SetStatusCondition(&gwc.Status.Conditions, metav1.Condition{
		    Type:   string(gateway.GatewayClassConditionStatusAccepted),
		    Status: "True",
		    Reason: string(gateway.GatewayClassReasonAccepted)})
	}
	err = r.client.Status().Update(ctx, gwc)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.GatewayClass{}).
		Complete(r)
}
