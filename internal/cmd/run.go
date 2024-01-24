/*
Copyright 2024 The Ketches Authors.

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

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/ketches/registry-proxy/internal/config"
	"github.com/ketches/registry-proxy/internal/global"
	"github.com/ketches/registry-proxy/pkg/image"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	Init()
}

// Run starts the registry-proxy admission webhook.
func Run() {
	http.HandleFunc(global.WebhookServicePath, mutatePod)
	log.Println("Start serving registry-proxy admission webhook ...")

	if err := http.ListenAndServeTLS(":443", global.WebhookServiceTLSCertFile, global.WebhookServiceTLSKeyFile, nil); err != nil {
		log.Fatalln("Failed to listen and serve admission webhook.", err.Error())
	}
}

// mutatePod is the handler of the admission webhook.
func mutatePod(w http.ResponseWriter, r *http.Request) {
	log.Println("Request admission webhook mutating ...")
	var (
		reviewRequest, reviewResponse admissionv1.AdmissionReview
		pod                           = &corev1.Pod{}
	)

	if err := json.NewDecoder(r.Body).Decode(&reviewRequest); err != nil {
		log.Println("Decode body failed.")
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	raw := reviewRequest.Request.Object.Raw

	if err := json.Unmarshal(raw, pod); err != nil {
		log.Println("Unmarshal pod object failed.", err.Error())
		http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
		return
	}

	reviewResponse.TypeMeta = reviewRequest.TypeMeta
	reviewResponse.Response = &admissionv1.AdmissionResponse{
		UID:     reviewRequest.Request.UID,
		Allowed: true,
		Result:  nil,
	}

	podName := pod.Name
	if podName == "" {
		// pod is controlled by a controller, use generateName instead
		podName = pod.GenerateName
	}

	log.Printf("Pod %s/%s is included", pod.Namespace, podName)

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
		log.Println("Marshal patch failed.")
		http.Error(w, fmt.Sprintf("could not marshal patch: %v", err), http.StatusInternalServerError)
		return
	}

	reviewResponse.Response.Patch = patchBytes
	reviewResponse.Response.PatchType = func() *admissionv1.PatchType {
		pt := admissionv1.PatchTypeJSONPatch
		return &pt
	}()
	reviewResponse.Response.Result = &metav1.Status{
		Message: fmt.Sprintf("registries in pod %s/%s is proxied", pod.Namespace, podName),
	}

	if err := json.NewEncoder(w).Encode(reviewResponse); err != nil {
		log.Println("Encode response failed.")
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// replaceImage replaces the image in the pod with the proxy image.
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

// getProxyImage gets the proxy image of the raw image.
func getProxyImage(rawImage string) string {
	var result = rawImage
	registry, name, err := image.Parse(rawImage)
	if err != nil {
		log.Println("Parse image failed.")
		return result
	}

	result = path.Join(getProxyRegistry(registry), name)
	if result != rawImage {
		log.Println(fmt.Sprintf("Proxy image: %s -> %s", rawImage, result))
	}
	return result
}

// getProxyRegistry gets the proxy registry of the raw registry.
func getProxyRegistry(rawRegistry string) string {
	proxies := config.GetProxies()
	if proxies == nil {
		return rawRegistry
	}
	newRegistry := proxies[rawRegistry]
	if newRegistry == "" {
		return rawRegistry
	}
	return newRegistry
}
