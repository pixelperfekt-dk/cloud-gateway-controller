package controllers

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const gateway_manifest string = `
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: foo-gateway
  namespace: foo-gateway-ns
spec:
  gatewayClassName: default
  listeners:
  - name: prod-web
    port: 80
    protocol: HTTP
    hostname: example.com`

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

var _ = Describe("Gateway controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a gateway", func() {
		It("Should create shadow gateway", func() {
			//By("xxx")
			ctx := context.Background()

			gw := &gateway.Gateway{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-gateway", Namespace: "default"},
				Spec: gateway.GatewaySpec{
					GatewayClassName: "cloud-gw",
					Listeners:        []gateway.Listener{gateway.Listener{Name: "prod-web", Port: 80, Protocol: "HTTP"}},
				},
			}
			Expect(k8sClient.Create(ctx, gw)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: gw.ObjectMeta.Name + "-istio", Namespace: gw.ObjectMeta.Namespace}
			createdGw := &gateway.Gateway{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdGw)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
