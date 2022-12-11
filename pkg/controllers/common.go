package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"

)

const (
	SelfControllerName gateway.GatewayController = "github.com/pixelperfekt-dk/cloud-gateway-controller"
)

type Controller interface {
	Client()        client.Client
	DynamicClient() dynamic.Interface
}

func patch(r Controller, ctx context.Context, us *unstructured.Unstructured, namespace string) error {
	log := log.FromContext(ctx)
	gvr, err := unstructuredToGVR(r, ctx, us)
	if err != nil {
		log.Error(err, "Cannot convert unstructured to GVR")
		return err
	}
	jsondata, err := json.Marshal(us.Object)
	if err != nil {
		log.Error(err, "Cannot convert unstructured to json")
		return err
	}

	log.Info("ApplyPatch", "jsondata", string(jsondata))
	c := r.DynamicClient().Resource(*gvr).Namespace(namespace)
	t := true
	_, err = c.Patch(ctx, us.GetName(), types.ApplyPatchType, jsondata, metav1.PatchOptions{
		Force:        &t,
		FieldManager: string(SelfControllerName),
	})
	return err
}

func unstructuredToGVR(r Controller, ctx context.Context, u *unstructured.Unstructured) (*schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(u.GetAPIVersion())
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{
		Group: gv.Group,
		Kind:  u.GetKind(),
	}
	mapping, err := r.Client().RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, err
	}
	return &schema.GroupVersionResource{
		Group:    gk.Group,
		Version:  gv.Version,
		Resource: mapping.Resource.Resource,
	}, nil
}

func renderTemplate(r Controller, gwParent *gateway.Gateway, configmap *corev1.ConfigMap, configmapKey string) (*unstructured.Unstructured, error) {
	var buf bytes.Buffer
	tmpl, found := configmap.Data[configmapKey]
	if !found {
		// TODO return error
		return nil, nil
	}
	ptmpl, err := template.New("tpl").Parse(tmpl)
	if err != nil {
		// TODO log
		return nil, err
	}
	input := &albTemplateValues{gwParent}
	err = ptmpl.Execute(io.Writer(&buf), input)
	if err != nil {
		// TODO log
		return nil, err
	}
	data := map[string]any{}
	err = yaml.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		return nil, err
	}
	us := unstructured.Unstructured{Object: data}
	return &us, nil
}
