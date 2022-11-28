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

## Building

```
make build container
make gateway-api-upstream-get
```

## Deploying

Setup test environment (using Istio for the 'shadow' gateway-class):

```
make create-cluster deploy-gateway-api deploy-istio deploy-contour
```

Deploy controller:

```
make kind-load-image deploy-controller
```

Deploy `GatewayClass` and a `ConfigMap` referenced by the `GatewayClass`. This
provides configuration for the controller:

```
kubectl apply -f test-data/gateway-class.yaml
```

Deploy an example `Gateway` and `HTTPRoute`:
 
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

*** The following is still work in progress

```
kubectl apply -f test-data/wip.yaml
```

*** End WIP section

Test access to test application:

```
curl -H 'Host: example.com' localhost/foo
```

Expect to see a `foo-bar` being echo'ed.
