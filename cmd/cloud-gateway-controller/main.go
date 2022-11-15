package main

import (
	"io"
	"log"
	"net/http"

	"github.com/michaelvl/cloud-gateway-controller/pkg/version"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"
	//kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// type Controller struct {
// 	metav1.TypeMeta   `json:",inline"`
// 	metav1.ObjectMeta `json:"metadata"`
// 	Spec              ControllerSpec   `json:"spec"`
// 	Status            ControllerStatus `json:"status"`
// }

// type ControllerSpec struct {
// 	Message string `json:"message"`
// }

type ControllerStatus struct {
	Succeeded int `json:"succeeded"`
}

type SyncRequest struct {
	Parent   gateway.Gateway     `json:"parent"`
	Children SyncRequestChildren `json:"children"`
}

type SyncRequestChildren struct {
	Gateways map[string]*gateway.Gateway `json:"Gateway.v1beta1"`
}

type SyncResponse struct {
	Status   ControllerStatus `json:"status"`
	Children []runtime.Object `json:"children"`
}

func sync(request *SyncRequest) (*SyncResponse, error) {
	response := &SyncResponse{}
	log.Printf("Got request %+v", request)

	gw_in := request.Parent
	log.Printf("Got %+v", gw_in)
	if gw_in.Spec.GatewayClassName == "foo-lb" {
		gw_out := gw_in.DeepCopy()
		gw_out.ResourceVersion = ""
		//gw_out := &gateway.Gateway{}
		gw_out.ObjectMeta.Name = gw_out.ObjectMeta.Name + "-istio"
		gw_out.Spec.GatewayClassName = "istio"
		response.Children = append(response.Children, gw_out)
	}
	return response, nil
}

func isOurGatewayClass(gw *gateway.Gateway) bool {
	if gw.Spec.GatewayClassName == "foo" {
		log.Printf("Gateway Class: %q\n", gw.Spec.GatewayClassName)
		return true
	}
	return false
}

// func sync(body []byte) (*gateway.Gateway, error) {
// 	gw := gateway.Gateway{}
// 	if err := kyaml.Unmarshal(body, gw); err != nil {
// 		return nil, err
// 	}
// 	return &gw, nil
// }

func syncHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	request := &SyncRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	response, err := sync(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, err = json.Marshal(&response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func main() {
	log.Printf("version: %s\n", version.Version)
	http.HandleFunc("/sync", syncHandler)
	http.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
