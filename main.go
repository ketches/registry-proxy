/*
Copyright 2023 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/ketches/registry-proxy/pkg/image"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"os"
	"path"
)

const (
	certFile = "/etc/webhook/certs/tls.crt"
	keyFile  = "/etc/webhook/certs/tls.key"

	matchAnnotationKey = "registry-proxy/enabled"
)

func init() {
	_, err := os.Stat(certFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = os.Stat(keyFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func main() {
	http.HandleFunc("/mutate", mutatePod)
	klog.Info("start serving registry-proxy admission webhook ...")
	if err := http.ListenAndServeTLS(":443", certFile, keyFile, nil); err != nil {
		log.Fatalln("failed to listen and serve admission webhook.", err.Error())
	}
}

func mutatePod(w http.ResponseWriter, r *http.Request) {
	log.Println("request admission webhook mutating ...")
	var (
		reviewRequest, reviewResponse v1.AdmissionReview
		pod                           = &corev1.Pod{}
	)

	if err := json.NewDecoder(r.Body).Decode(&reviewRequest); err != nil {
		log.Println("decode body failed.")
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	raw := reviewRequest.Request.Object.Raw

	if err := json.Unmarshal(raw, pod); err != nil {
		log.Println("unmarshal pod object failed.", err.Error())
		http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
		return
	}

	reviewResponse.TypeMeta = reviewRequest.TypeMeta
	reviewResponse.Response = &v1.AdmissionResponse{
		UID:     reviewRequest.Request.UID,
		Allowed: true,
		Result:  nil,
		PatchType: func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	if pod.Annotations[matchAnnotationKey] == "true" {
		fmt.Printf("pod %s/%s is matched\n", pod.Namespace, pod.Name)
		reviewResponse.Response.Result = &metav1.Status{
			Message: fmt.Sprintf("pod %s/%s is matched", pod.Namespace, pod.Name),
		}

		replaceImage(pod)

		var patches = []map[string]any{
			{
				"op":    "replace",
				"path":  "/spec/initContainers",
				"value": pod.Spec.InitContainers,
			},
			{
				"op":    "replace",
				"path":  "/spec/containers",
				"value": pod.Spec.Containers,
			},
		}

		patchBytes, err := json.Marshal(patches)
		if err != nil {
			log.Println("marshal patch failed.")
			http.Error(w, fmt.Sprintf("could not marshal patch: %v", err), http.StatusInternalServerError)
			return
		}

		reviewResponse.Response.Patch = patchBytes
	}

	if err := json.NewEncoder(w).Encode(reviewResponse); err != nil {
		log.Println("encode response failed.")
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func replaceImage(pod *corev1.Pod) {
	for i, _ := range pod.Spec.InitContainers {
		pod.Spec.InitContainers[i].Image = getProxyImage(pod.Spec.InitContainers[i].Image)
	}

	for i, _ := range pod.Spec.Containers {
		pod.Spec.Containers[i].Image = getProxyImage(pod.Spec.Containers[i].Image)
	}

	for i, _ := range pod.Spec.EphemeralContainers {
		pod.Spec.EphemeralContainers[i].EphemeralContainerCommon.Image = getProxyImage(pod.Spec.EphemeralContainers[i].EphemeralContainerCommon.Image)
	}
}

func getProxyImage(rawImage string) string {
	registry, name, err := image.Parse(rawImage)
	if err != nil {
		log.Println("parse image failed.")
		return rawImage
	}

	return path.Join(getProxyRegistry(registry), name)
}

func getProxyRegistry(rawRegistry string) string {
	newRegistry := registries[rawRegistry]
	if newRegistry == "" {
		return rawRegistry
	}

	return newRegistry
}

var registries = map[string]string{
	"docker.io":         "dockerproxy.com",
	"ghcr.io":           "ghcr.dockerproxy.com",
	"gcr.io":            "gcr.dockerproxy.com",
	"k8s.gcr.io":        "k8s.dockerproxy.com",
	"registry.k8s.io":   "k8s.dockerproxy.com",
	"quay.io":           "quay.dockerproxy.com",
	"mcr.microsoft.com": "mcr.dockerproxy.com",
}
