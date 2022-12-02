package controllers

import (
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	"testing"
)

const gateway_manifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway
spec:
  gatewayClassName: default
  listeners:
  - name: prod-web
    port: 80
    protocol: HTTP
    hostname: example.com`

const gatewayclass_manifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: GatewayClass
metadata:
  name: default
spec:
  controllerName: "github.com/pixelperfekt-dk/cloud-gateway-controller"
  parametersRef:
    group: v1
    kind: ConfigMap
    name: default-gateway-class`

const configmap_manifest string = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: default-gateway-class
  namespace: default
data:
  tier2GatewayClass: istio`

func TestGatewayClass(t *testing.T) {
	r := GatewayReconciler{}
	gw := &gateway.Gateway{}
	cm := &corev1.ConfigMap{}
	_ = yaml.Unmarshal([]byte(gateway_manifest), gw)
	_ = yaml.Unmarshal([]byte(configmap_manifest), cm)
	gw_out, err := r.constructGateway(gw, cm)
	if err != nil {
		t.Fatalf("Error converting gateway: %+v, %q", gw, err)
	}
	if gw_out == nil {
		t.Fatalf("Error converting gateway: %+v, %q", gw, err)
	}
	if gw_out.Spec.GatewayClassName != "istio" {
		t.Errorf("Unexpected GatewayClassName: %+v", gw_out)
	}
}
