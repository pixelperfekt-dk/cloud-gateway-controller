package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	//"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases"), filepath.Join("..", "..", "upstream-gateway-api-crds")},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = gateway.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Setup GatewayClass
	cm := &v1.ConfigMap{}
	cmdata, err := os.ReadFile("../../test-data/gateway-class-configmap.yaml")
	Expect(err).ToNot(HaveOccurred())
	_ = yaml.Unmarshal(cmdata, cm)
	Expect(k8sClient.Create(ctx, cm)).Should(Succeed())

	gwc := &gateway.GatewayClass{}
	gwcdata, err := os.ReadFile("../../test-data/gateway-class.yaml")
	Expect(err).ToNot(HaveOccurred())
	_ = yaml.Unmarshal(gwcdata, gwc)
	Expect(k8sClient.Create(ctx, gwc)).Should(Succeed())

	// Create controllers
	gwcctrl := NewGatewayClassController(mgr)
	err = gwcctrl.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	gwctrl := NewGatewayController(mgr)
	err = gwctrl.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	rtctrl := NewHTTPRouteController(mgr)
	err = rtctrl.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
