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

type HTTPRouteReconciler struct {
	client.Client
	dynamicClient dynamic.Interface
	scheme        *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io.tutorial.kubebuilder.io,resources=httproutes/finalizers,verbs=update

func NewHTTPRouteController(mgr ctrl.Manager) *HTTPRouteReconciler {
	r := &HTTPRouteReconciler{
		Client:        mgr.GetClient(),
		dynamicClient: dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie()),
		scheme:        mgr.GetScheme(),
	}
	return r
}

func (r *HTTPRouteReconciler) GetClient() client.Client {
	return r.Client
}

func (r *HTTPRouteReconciler) DynamicClient() dynamic.Interface {
	return r.dynamicClient
}

func (r *HTTPRouteReconciler) Scheme() *runtime.Scheme {
	return r.scheme
}

func (r *HTTPRouteReconciler) constructHTTPRoute(rtIn *gateway.HTTPRoute, configmap *corev1.ConfigMap) (*gateway.HTTPRoute, error) {
	name := fmt.Sprintf("%s-%s", rtIn.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	rtOut := rtIn.DeepCopy()
	rtOut.ResourceVersion = ""
	rtOut.ObjectMeta.Name = name
	// FIXME, should follow pattern in gateway-controller and remap parents of type gateway similarly
	rtOut.Spec.CommonRouteSpec.ParentRefs[0].Name = "foo-gateway-istio"

	return rtOut, nil
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	rt := &gateway.HTTPRoute{}
	err := r.Get(ctx, req.NamespacedName, rt)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconcile", "httproute", rt)

	prefs := rt.Spec.CommonRouteSpec.ParentRefs
	// FIXME check kind of parent ref is Gateway and missing parentRef. Accepts more than one parent ref
	pref := prefs[0]

	gw := &gateway.Gateway{}
	err = r.Get(ctx, types.NamespacedName{Name: string(pref.Name), Namespace: string(*pref.Namespace)}, gw)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconcile", "gateway", gw)

	gwclass, configmap, err := lookupGatewayClass(ctx, r, string(gw.Spec.GatewayClassName))
	if err != nil {
		return ctrl.Result{}, err
	} else if gwclass == nil || configmap == nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create HTTPRoute resource
	rtOut, err := r.constructHTTPRoute(rt, configmap)
	if err != nil {
		log.Error(err, "unable to build HTTPRoute object", "httproute", rt)
		return ctrl.Result{}, err
	}

	log.Info("create httproute", "rtOut", rtOut)

	if err := ctrl.SetControllerReference(rt, rtOut, r.Scheme()); err != nil {
		log.Error(err, "unable to set controllerreference for httproute", "rtOut", rtOut)
		return ctrl.Result{}, err
	}

	rtFound := &gateway.HTTPRoute{}
	err = r.Get(ctx, types.NamespacedName{Name: rtOut.Name, Namespace: rtOut.Namespace}, rtFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("create gateway")
		if err := r.Create(ctx, rtOut); err != nil {
			log.Error(err, "unable to create Gateway", "httproute", rtOut)
			return ctrl.Result{}, err
		}
	} else if err == nil {
		rtFound.Spec = rtOut.Spec
		log.Info("update httproute", "rt", rtFound)
		if err := r.Update(ctx, rtFound); err != nil {
			log.Error(err, "unable to update HTTPRoute", "httproute", rtFound)
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
