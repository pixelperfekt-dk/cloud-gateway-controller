package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type GatewayReconciler struct {
	client        client.Client
	dynamicClient dynamic.Interface
	scheme        *runtime.Scheme
}

type albTemplateValues struct {
	//Tier1 *gateway.Gateway
	*gateway.Gateway
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/finalizers,verbs=update

func NewGatewayController(mgr ctrl.Manager) *GatewayReconciler {
	r := &GatewayReconciler{
		client:        mgr.GetClient(),
		dynamicClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		scheme:        mgr.GetScheme(),
	}
	return r
}

func (r *GatewayReconciler) Client() client.Client {
	return r.client
}

func (r *GatewayReconciler) DynamicClient() dynamic.Interface {
	return r.dynamicClient
}

func (r *GatewayReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *GatewayReconciler) constructGateway(gwIn *gateway.Gateway, configmap *corev1.ConfigMap) (*gateway.Gateway, error) {
	name := fmt.Sprintf("%s-%s", gwIn.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	gwOut := gwIn.DeepCopy()
	gwOut.ResourceVersion = ""
	gwOut.ObjectMeta.Name = name
	if gwOut.ObjectMeta.Annotations == nil {
		gwOut.ObjectMeta.Annotations = map[string]string{}
	}
	gwOut.ObjectMeta.Annotations["networking.istio.io/service-type"] = "ClusterIP"
	gwOut.Spec.GatewayClassName = gateway.ObjectName(configmap.Data["tier2GatewayClass"])

	return gwOut, nil
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	gw := &gateway.Gateway{}
	err := r.client.Get(ctx, req.NamespacedName, gw)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconcile", "gateway", gw)

	// Lookup class and configuration
	gwclass, configmap, err := lookupGatewayClass(ctx, r, string(gw.Spec.GatewayClassName))
	if err != nil {
		return ctrl.Result{}, err
	} else if gwclass == nil || configmap == nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create Gateway resource
	gwOut, err := r.constructGateway(gw, configmap)
	if err != nil {
		log.Error(err, "unable to build Gateway object", "gateway", gw)
		return ctrl.Result{}, err
	}

	log.Info("create gateway", "gwOut", gwOut)

	if err := ctrl.SetControllerReference(gw, gwOut, r.Scheme()); err != nil {
		log.Error(err, "unable to set controllerreference for gateway", "gwOut", gwOut)
		return ctrl.Result{}, err
	}

	gwFound := &gateway.Gateway{}
	err = r.client.Get(ctx, types.NamespacedName{Name: gwOut.Name, Namespace: gwOut.Namespace}, gwFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("create gateway")
		if err := r.client.Create(ctx, gwOut); err != nil {
			log.Error(err, "unable to create Gateway", "gateway", gwOut)
			return ctrl.Result{}, err
		}
	} else if err == nil {
		gwFound.Spec = gwOut.Spec
		log.Info("update gateway", "gw", gwFound)
		if err := r.client.Update(ctx, gwFound); err != nil {
			log.Error(err, "unable to update Gateway", "gateway", gwFound)
			return ctrl.Result{}, err
		}
	}

	// Create ALB resource
	err = createUpdateFromTemplate(ctx, r, gw, configmap, "albTemplate")
	if err != nil {
		log.Error(err, "unable to build alb object", "gateway", gw)
		return ctrl.Result{}, err
	}

	// Create TLS certificate resource
	err = createUpdateFromTemplate(ctx, r, gw, configmap, "tlsCertificateTemplate")
	if err != nil {
		log.Error(err, "unable to build certificate object", "gateway", gw)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.Gateway{}).
		Owns(&gateway.Gateway{}). // FIXME, more types
		Complete(r)
}
