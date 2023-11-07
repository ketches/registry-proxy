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
	"log"
	"net/http"
	"os"
	"path"
	"slices"

	"github.com/fatih/color"
	"github.com/ketches/registry-proxy/pkg/conf"
	"github.com/ketches/registry-proxy/pkg/image"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	certFile = "/etc/webhook/certs/tls.crt"
	keyFile  = "/etc/webhook/certs/tls.key"
)

func init() {
	fmt.Println(color.GreenString("Welcome to use registry-proxy!"))
	conf.HotLoading()
}

func main() {
	http.HandleFunc("/mutate", mutatePod)
	log.Println("Start serving registry-proxy admission webhook ...")
	if os.Getenv("LOCAL_DEBUG") == "true" {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalln("failed to listen and serve admission webhook.", err.Error())
		}
	} else {
		if err := http.ListenAndServeTLS(":443", certFile, keyFile, nil); err != nil {
			log.Fatalln("failed to listen and serve admission webhook.", err.Error())
		}
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
	}

	podName := pod.Name
	if podName == "" {
		// pod is controlled by a controller, use generateName instead
		podName = pod.GenerateName
	}

	if isPodIncluded(pod) {
		log.Printf("pod %s/%s is included", pod.Namespace, podName)

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
		reviewResponse.Response.PatchType = func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
			return &pt
		}()
		reviewResponse.Response.Result = &metav1.Status{
			Message: fmt.Sprintf("registries in pod %s/%s is proxied", pod.Namespace, podName),
		}
	} else {
		log.Printf("pod %s/%s is excluded", pod.Namespace, podName)
	}

	if err := json.NewEncoder(w).Encode(reviewResponse); err != nil {
		log.Println("encode response failed.")
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func replaceImage(pod *corev1.Pod) {
	for i := range pod.Spec.InitContainers {
		pod.Spec.InitContainers[i].Image = getProxyImage(pod.Spec.InitContainers[i].Image)
	}

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Image = getProxyImage(pod.Spec.Containers[i].Image)
	}

	for i := range pod.Spec.EphemeralContainers {
		pod.Spec.EphemeralContainers[i].EphemeralContainerCommon.Image = getProxyImage(pod.Spec.EphemeralContainers[i].EphemeralContainerCommon.Image)
	}
}

func getProxyImage(rawImage string) string {
	var result = rawImage
	registry, name, err := image.Parse(rawImage)
	if err != nil {
		log.Println("parse image failed.")
		return result
	}

	result = path.Join(getProxyRegistry(registry), name)
	if result != rawImage {
		log.Println(color.CyanString("proxy image: " + color.YellowString(rawImage) + " -> " + color.HiGreenString(result)))
	}
	return result
}

func getProxyRegistry(rawRegistry string) string {
	proxies := conf.GetProxies()
	if proxies == nil {
		return rawRegistry
	}
	newRegistry := proxies[rawRegistry]
	if newRegistry == "" {
		return rawRegistry
	}
	return newRegistry
}

func isPodIncluded(pod *corev1.Pod) bool {
	excludeNamespaces := conf.GetExcludeNamespaces()
	if slices.Contains(excludeNamespaces, "*") || slices.Contains(excludeNamespaces, pod.Namespace) {
		return false
	}

	includeNamespaces := conf.GetIncludeNamespaces()
	return slices.Contains(includeNamespaces, "*") || slices.Contains(includeNamespaces, pod.Namespace)
}
