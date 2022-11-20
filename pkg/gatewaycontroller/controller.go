package gatewaycontroller

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	//lister "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
)

type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/finalizers,verbs=update

func (r *GatewayReconciler) constructGateway(gw_in *gateway.Gateway, class gateway.GatewayClass) (*gateway.Gateway, error) {
	name := fmt.Sprintf("%s-%s", gw_in.ObjectMeta.Name, "istio") // FIXME create suffix from gatewayclass configmap
	gw_out := gw_in.DeepCopy()
	gw_out.ResourceVersion = ""
	gw_out.ObjectMeta.Name = name
	gw_out.Spec.GatewayClassName = "istio" // FIXME get from gatewayclass configmap

	if err := ctrl.SetControllerReference(gw_in, gw_out, r.Scheme); err != nil {
		return nil, err
        }
	return gw_out, nil
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	gw := &gateway.Gateway{}
	err := r.Get(ctx, req.NamespacedName, gw)
	if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("reconsile", "gateway", gw)

	classes := &gateway.GatewayClassList{}
	err = r.List(ctx, classes, client.InNamespace(metav1.NamespaceAll))
	if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("reconsile", "gatewayclasses", classes)

	if gw.Spec.GatewayClassName == "default" {
		log.Info("handled by us", "gatewayclass", gw.Spec.GatewayClassName)
		// FIXME test not already existing
		gw_out, err := r.constructGateway(gw, classes.Items[0]) // FIXME, find matching class
		if err != nil {
			log.Error(err, "unable to build Gateway object", "gateway", gw)
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, gw_out); err != nil {
			log.Error(err, "unable to create Gateway", "gateway", gw)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.Gateway{}).
		Complete(r)
}
