package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	"testing"
)

// const gateway_manifest string = `apiVersion: gateway.networking.k8s.io/v1beta1
// kind: Gateway
// metadata:
//   name: foo-gateway
//   namespace: gateway-api-example-ns1
// spec:
//   gatewayClassName: foo-lb`

// func TestSync(t *testing.T) {
// 	gw, err := sync([]byte(gateway_manifest))
// 	if err != nil || gw.ObjectMeta.Name != "foo-gateway" {
// 		t.Errorf("Error testing API: err %q gw %+v", err, gw.ObjectMeta.Name)
// 	}
// }

func TestGatewayClass(t *testing.T) {
	gw := gateway.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "foo-ns",
		},
		Spec: gateway.GatewaySpec{
			GatewayClassName: "foo",
		},
	}
	isOurs := isOurGatewayClass(&gw)
	if !isOurs {
		t.Errorf("Error testing API: %+v", gw)
	}
}
