package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	//"k8s.io/apiextensions-apiserver/pkg/client/clientset/deprecated/scheme"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func (r *GatewayReconciler) constructGateway(gw_in *gateway.Gateway, configmap *corev1.ConfigMap) (*gateway.Gateway, error) {
	name := fmt.Sprintf("%s-%s", gw_in.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	gw_out := gw_in.DeepCopy()
	gw_out.ResourceVersion = ""
	gw_out.ObjectMeta.Name = name
	if gw_out.ObjectMeta.Annotations == nil {
		gw_out.ObjectMeta.Annotations = map[string]string{}
	}
	gw_out.ObjectMeta.Annotations["networking.istio.io/service-type"] = "ClusterIP"
	gw_out.Spec.GatewayClassName = gateway.ObjectName(configmap.Data["tier2GatewayClass"])

	return gw_out, nil
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
	gwclass, configmap, err := lookupGatewayClass(r, ctx, string(gw.Spec.GatewayClassName))
	if err != nil {
		return ctrl.Result{}, err
	} else if gwclass == nil || configmap == nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create Gateway resource
	gw_out, err := r.constructGateway(gw, configmap)
	if err != nil {
		log.Error(err, "unable to build Gateway object", "gateway", gw)
		return ctrl.Result{}, err
	}

	log.Info("create gateway", "gw_out", gw_out)

	if err := ctrl.SetControllerReference(gw, gw_out, r.Scheme()); err != nil {
		log.Error(err, "unable to set controllerreference for gateway", "gw_out", gw_out)
		return ctrl.Result{}, err
	}

	gw_found := &gateway.Gateway{}
	err = r.client.Get(ctx, types.NamespacedName{Name: gw_out.Name, Namespace: gw_out.Namespace}, gw_found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("create gateway")
		if err := r.client.Create(ctx, gw_out); err != nil {
			log.Error(err, "unable to create Gateway", "gateway", gw_out)
			return ctrl.Result{}, err
		}
	} else if err == nil {
		gw_found.Spec = gw_out.Spec
		log.Info("update gateway", "gw", gw_found)
		if err := r.client.Update(ctx, gw_found); err != nil {
			log.Error(err, "unable to update Gateway", "gateway", gw_found)
			return ctrl.Result{}, err
		}
	}

	// Create ALB resource
	err = createUpdateFromTemplate(r, ctx, gw, configmap, "albTemplate")
	if err != nil {
		log.Error(err, "unable to build alb object", "gateway", gw)
		return ctrl.Result{}, err
	}
	// alb_u, err := renderTemplate(r, gw, configmap, "albTemplate")
	// if err != nil {
	// 	log.Error(err, "unable to set render alb template")
	// 	return ctrl.Result{}, err
	// }

	// log.Info("create alb", "alb_u", alb_u)

	// if err := ctrl.SetControllerReference(gw, alb_u, r.Scheme()); err != nil {
	// 	log.Error(err, "unable to set controllerreference for alb template", "alb_u", alb_u)
	// 	return ctrl.Result{}, err
	// }

	// if err := patch(r, ctx, alb_u, gw.ObjectMeta.Namespace); err != nil {
	// 	log.Error(err, "unable to patch", "alb_u", alb_u)
	// 	return ctrl.Result{}, err
	// }

	// Create TLS certificate resource
	err = createUpdateFromTemplate(r, ctx, gw, configmap, "tlsCertificateTemplate")
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
