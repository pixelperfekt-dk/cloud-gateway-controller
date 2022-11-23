package gatewaycontroller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	//lister "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
)

const (
	SelfControllerName gateway.GatewayController = "github.com/pixelperfekt-dk/cloud-gateway-controller"
)

type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/finalizers,verbs=update

func (r *GatewayReconciler) constructGateway(gw_in *gateway.Gateway, configmap *corev1.ConfigMap) (*gateway.Gateway, error) {
	name := fmt.Sprintf("%s-%s", gw_in.ObjectMeta.Name, "istio") // FIXME create suffix from gatewayclass configmap
	gw_out := gw_in.DeepCopy()
	gw_out.ResourceVersion = ""
	gw_out.ObjectMeta.Name = name
	gw_out.Spec.GatewayClassName = gateway.ObjectName(configmap.Data["tier2GatewayClass"])

	return gw_out, nil
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	gw := &gateway.Gateway{}
	err := r.Get(ctx, req.NamespacedName, gw)
	if err != nil {
		return reconcile.Result{}, err
	}
	log.Info("reconcile", "gateway", gw)

	classes := &gateway.GatewayClassList{}
	err = r.List(ctx, classes, client.InNamespace(metav1.NamespaceAll))
	if err != nil {
		return reconcile.Result{}, err
	}

	// Look for a matching GatewayClass that are implemented by us
	var gwclass *gateway.GatewayClass
	for _, gwc := range classes.Items {
		if gwc.Spec.ControllerName == SelfControllerName && gwc.ObjectMeta.Name == string(gw.Spec.GatewayClassName) {
			log.Info("defined by", "gatewayclass", gwc)
			gwclass = &gwc
			break
		}
	}

	if gwclass != nil {
		log.Info("reconcile", "gatewayclasses", gwclass)
		// Lookup associated ConfigMap
		var configmap *corev1.ConfigMap
		if gwclass.Spec.ParametersRef != nil {
			configmaps := &corev1.ConfigMapList{}
			// TODO: Check parametersref group+kind is configmap
			err = r.List(ctx, configmaps, client.InNamespace(string(*gwclass.Spec.ParametersRef.Namespace)))
			if err != nil {
				return reconcile.Result{}, err
			}
			for _, cm := range configmaps.Items {
				if cm.ObjectMeta.Name == gwclass.Spec.ParametersRef.Name {
					log.Info("defined by", "configmap", cm)
					configmap = &cm
					break
				}
			}
		}
		log.Info("configured by", "configmap", configmap.Data)

		// FIXME test not already existing
		gw_out, err := r.constructGateway(gw, configmap)
		if err != nil {
			log.Error(err, "unable to build Gateway object", "gateway", gw)
			return ctrl.Result{}, err
		}
		if err := ctrl.SetControllerReference(gw, gw_out, r.Scheme); err != nil {
			log.Error(err, "unable to set controllerreference for Gateway", "gateway", gw_out)
			return ctrl.Result{}, err
		}

		gw := &gateway.Gateway{}
		err = r.Get(ctx, req.NamespacedName, gw)
		if err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, gw_out); err != nil {
				log.Error(err, "unable to create Gateway", "gateway", gw)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// TODO: Compare, if not equal, update
			if err := r.Update(ctx, gw_out); err != nil {
				log.Error(err, "unable to create Gateway", "gateway", gw)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gateway.Gateway{}).
		Complete(r)
}
