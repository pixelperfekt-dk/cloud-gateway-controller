package controllers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
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

type HTTPRouteReconciler struct {
	client        client.Client
	dynamicClient dynamic.Interface
	scheme        *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/finalizers,verbs=update

func NewHTTPRouteController(mgr ctrl.Manager) *HTTPRouteReconciler {
	r := &HTTPRouteReconciler{
		client:        mgr.GetClient(),
		dynamicClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		scheme:        mgr.GetScheme(),
	}
	return r
}

func (r *HTTPRouteReconciler) Client() client.Client {
	return r.client
}

func (r *HTTPRouteReconciler) DynamicClient() dynamic.Interface {
	return r.dynamicClient
}

func (r *HTTPRouteReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *HTTPRouteReconciler) constructHTTPRoute(rt_in *gateway.HTTPRoute, configmap *corev1.ConfigMap) (*gateway.HTTPRoute, error) {
	name := fmt.Sprintf("%s-%s", rt_in.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	rt_out := rt_in.DeepCopy()
	rt_out.ResourceVersion = ""
	rt_out.ObjectMeta.Name = name
	// FIXME, should follow pattern in gateway-controller and remap parents of type gateway similarly
	rt_out.Spec.CommonRouteSpec.ParentRefs[0].Name = "foo-gateway-istio"

	return rt_out, nil
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	rt := &gateway.HTTPRoute{}
	err := r.client.Get(ctx, req.NamespacedName, rt)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// FIXME, need to check with ParentReference if this is our HTTPRoute
	if strings.HasSuffix(rt.ObjectMeta.Name, "-istio") {
		return ctrl.Result{}, nil
	}

	log.Info("reconcile", "httproute", rt)

	// Lookup class and configuration
	//gwclass, configmap, err := lookupGatewayClass(r, ctx, gw.Spec.GatewayClassName)
	gwclass, configmap, err := lookupGatewayClass(r, ctx, "cloud-gw")   // FIXME
	if err != nil {
		return ctrl.Result{}, err
	} else if gwclass == nil || configmap == nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create HTTPRoute resource
	rt_out, err := r.constructHTTPRoute(rt, configmap)
	if err != nil {
		log.Error(err, "unable to build HTTPRoute object", "httproute", rt)
		return ctrl.Result{}, err
	}

	log.Info("create httproute", "rt_out", rt_out)

	if err := ctrl.SetControllerReference(rt, rt_out, r.Scheme()); err != nil {
		log.Error(err, "unable to set controllerreference for httproute", "rt_out", rt_out)
		return ctrl.Result{}, err
	}

	rt_found := &gateway.HTTPRoute{}
	err = r.client.Get(ctx, types.NamespacedName{Name: rt_out.Name, Namespace: rt_out.Namespace}, rt_found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("create gateway")
		if err := r.client.Create(ctx, rt_out); err != nil {
			log.Error(err, "unable to create Gateway", "httproute", rt_out)
			return ctrl.Result{}, err
		}
	} else if err == nil {
		rt_found.Spec = rt_out.Spec
		log.Info("update httproute", "rt", rt_found)
		if err := r.client.Update(ctx, rt_found); err != nil {
			log.Error(err, "unable to update HTTPRoute", "httproute", rt_found)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.HTTPRoute{}).
		Owns(&gateway.HTTPRoute{}).
		Complete(r)
}
