package gatewaycontroller

import (
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	"testing"
	"gopkg.in/yaml.v3"
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

// func TestSync(t *testing.T) {
// 	gw, err := sync([]byte(gateway_manifest))
// 	if err != nil || gw.ObjectMeta.Name != "foo-gateway" {
// 		t.Errorf("Error testing API: err %q gw %+v", err, gw.ObjectMeta.Name)
// 	}
// }

func TestGatewayClass(t *testing.T) {
	r := GatewayReconciler{}
	gw := &gateway.Gateway{}
	gwc := &gateway.GatewayClass{}
	_ = yaml.Unmarshal([]byte(gateway_manifest), gw)
	_ = yaml.Unmarshal([]byte(gatewayclass_manifest), gwc)
	gw_out,err := r.constructGateway(gw, gwc)
	if err!=nil {
		t.Errorf("Error converting gateway: %+v, %q", gw, err)
	}
	if gw_out==nil {
		t.Errorf("Error converting gateway: %+v, %q", gw, err)
	}
	if gw_out.Spec.GatewayClassName != "istio" {
		t.Errorf("Unexpected GatewayClassName: %+v", gw_out)
	}
}
