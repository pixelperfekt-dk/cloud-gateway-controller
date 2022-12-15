# Cloud Gateway Controller

The [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) is an API for
describing network gateways and configure routing from gateways to Kubernetes
services.

This repository contain a Kubernetes controller that implements the
`GatewayClass` and `Gateway` resources of the Gateway API. In response to these
resources it creates a 'shadow' `Gateway` resource with a different
gateway-class and other resources to implement a network infrastructure. Hence,
this controller is a controller of controllers. This concept is described in
[Future of Cloud LB
Integration](https://events.istio.io/istiocon-2022/slides/f3-K8sGatewayAPIs.pdf). Particularly
this slide:

> ![IstioCon slide](doc/images/istiocon-slide.png)

The controller have several similarities with [GKE Gateway
controller](https://cloud.google.com/kubernetes-engine/docs/concepts/gateway-api#gateway_controller),
except this controller aims at being cloud agnostic.

## Objective



## Why Not Use e.g. Crossplane or Helm?

An important objective of the cloud-gateway-controller is to maintain
a Gateway-API compatible interface towards users. This would not be
possible with techniques such as Crossplane and Helm.

## Building

```
make build container
make gateway-api-upstream-get
```

## Deploying

Setup test environment, which use Istio for the 'shadow'
gateway-class, Contour for the front load balancer and cert-manager to
issue TLS certificates:

```
make create-cluster deploy-gateway-api deploy-istio deploy-contour deploy-cert-manager
```

Deploy controller:

```
make kind-load-image deploy-controller
```

To watch the progress ans resources created, it can be convenient to watch for
resources with the following command:

```
watch kubectl get gateway,httproute,ingress,certificate,secret,po,gatewayclass
```

Deploy `GatewayClass` and a `ConfigMap` referenced by the `GatewayClass`. This
provides configuration for the controller:

```
kubectl apply -f test-data/gateway-class.yaml
```

Deploy an example `Gateway` and `HTTPRoute` with the following command. You can
review the `Gateway` and `HTTPRoute` resources by leaving out the `kubectl apply
-f -` part:

```
helm template foo gateway-api --repo https://pixelperfekt-dk.github.io/helm-charts --values test-data/user-gateway-values.yaml | kubectl apply -f -
```

In response to the `foo-gateway-api` `Gateway` created, expect to see a shadow
`Gateway` called `foo-gateway-api-istio`. Also, expect to see Istio respond to
the `foo-gateway-api-istio` `Gateway` by creating an ingress-gateway
deployment. The PODs created for the Istio ingress-gateway names will start with
`foo-gateway-api-istio-`.

Deploy a test application that matches the deployed `HTTPRoute`:

```
kubectl apply -f test-data/test-app.yaml
```

Test access to test application:

```
curl --resolve example.com:80:127.0.0.1 http://example.com/foo
```

Expect to see a `foo-bar` being echo'ed.

Similarly, but using HTTPS through the cert-manager issued
certificate:

```
curl --cacert example-com.crt --resolve example.com:443:127.0.0.1 https://example.com/foo
```
