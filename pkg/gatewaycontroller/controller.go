package gatewaycontroller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"
)

const (
	SelfControllerName gateway.GatewayController = "github.com/pixelperfekt-dk/cloud-gateway-controller"
)

type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type albTemplateValues struct {
	//Tier1 *gateway.Gateway
	*gateway.Gateway
}

//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/finalizers,verbs=update

func (r *GatewayReconciler) constructGateway(gw_in *gateway.Gateway, configmap *corev1.ConfigMap) (*gateway.Gateway, error) {
	name := fmt.Sprintf("%s-%s", gw_in.ObjectMeta.Name, configmap.Data["tier2GatewayClass"])
	gw_out := gw_in.DeepCopy()
	gw_out.ResourceVersion = ""
	gw_out.ObjectMeta.Name = name
	if gw_out.ObjectMeta.Annotations==nil {
		gw_out.ObjectMeta.Annotations = map[string]string{}
	}
	gw_out.ObjectMeta.Annotations["networking.istio.io/service-type"] = "ClusterIP"
	gw_out.Spec.GatewayClassName = gateway.ObjectName(configmap.Data["tier2GatewayClass"])

	return gw_out, nil
}

func (r *GatewayReconciler) constructAlbTemplate(gw_in, gw_out *gateway.Gateway, configmap *corev1.ConfigMap) ([]byte, error) {
	var b bytes.Buffer
	tmpl, found := configmap.Data["albTemplate"]
	if !found {
		return b.Bytes(), nil
	}
	ptmpl, err := template.New("alb").Parse(tmpl)
	if err != nil {
		// TODO log
		return b.Bytes(), nil
	}
	data := &albTemplateValues{gw_out}
	err = ptmpl.Execute(io.Writer(&b), data)
	if err != nil {
		// TODO log
		return b.Bytes(), nil
	}
	return b.Bytes(), nil
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

		// Create Gateway resource

		gw_found := &gateway.Gateway{}
		err = r.Get(ctx, types.NamespacedName{Name: gw_out.Name, Namespace: gw_out.Namespace}, gw_found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("create gateway")
			if err := r.Create(ctx, gw_out); err != nil {
				log.Error(err, "unable to create Gateway", "gateway", gw_out)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// TODO: Compare gw_out and gw_found, if not equal, update
			log.Info("update gateway", "gw", gw_found)
			if err := r.Update(ctx, gw_out); err != nil {
				log.Error(err, "unable to update Gateway", "gateway", gw_out)
				return ctrl.Result{}, err
			}
		}

		// Create ALB resource
		alb, err := r.constructAlbTemplate(gw, gw_out, configmap)
		log.Info("create alb", "alb", string(alb))
		ing := &netv1.Ingress{}
		if err := yaml.Unmarshal(alb, ing); err != nil {
		}
		if err := ctrl.SetControllerReference(gw, ing, r.Scheme); err != nil {
			log.Error(err, "unable to set controllerreference for Ingress", "ingress", ing)
			return ctrl.Result{}, err
		}

		ing_found := &netv1.Ingress{}
		err = r.Get(ctx, types.NamespacedName{Name: ing.Name, Namespace: ing.Namespace}, ing_found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("create ingress")
			if err := r.Create(ctx, ing); err != nil {
				log.Error(err, "unable to create Ingress", "inress", ing)
				return ctrl.Result{}, err
			}
		} else if err == nil {
			// TODO: Compare ing and ing_found, if not equal, update
			log.Info("update ingress", "ing", ing_found)
			if err := r.Update(ctx, ing); err != nil {
				log.Error(err, "unable to update Ingress", "ingress", ing)
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
