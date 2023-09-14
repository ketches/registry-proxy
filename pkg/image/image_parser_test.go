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

package image

import "testing"

func TestParseImage(t *testing.T) {
	testdata := []struct {
		image  string
		domain string
		name   string
	}{
		{
			image:  "stilleshan/frpc:latest",
			domain: "docker.io",
			name:   "docker.io/stilleshan/frpc:latest",
		},
		{
			image:  "nginx:latest",
			domain: "docker.io",
			name:   "docker.io/library/nginx:latest",
		},
		{
			image:  "ghcr.io/nginxinc/nginx-kubernetes-gateway:edge",
			domain: "ghcr.io",
			name:   "ghcr.io/nginxinc/nginx-kubernetes-gateway:edge",
		},
		{
			image:  "ghcr.io/nginxinc/nginx-kubernetes-gateway/nginx:edge",
			domain: "ghcr.io",
			name:   "ghcr.io/nginxinc/nginx-kubernetes-gateway/nginx:edge",
		},
		{
			image:  "registry.k8s.io/kube-apiserver:v1.28.0",
			domain: "registry.k8s.io",
			name:   "registry.k8s.io/kube-apiserver:v1.28.0",
		},
		{
			image:  "registry.k8s.io/gateway-api/admission-server:v0.8.0",
			domain: "registry.k8s.io",
			name:   "registry.k8s.io/gateway-api/admission-server:v0.8.0",
		},
	}

	for _, td := range testdata {
		domain, named, err := Parse(td.image)
		if err != nil {
			t.Errorf("parse image failed: %v", err)
		}
		if domain != td.domain {
			t.Errorf("parse image failed, expected domain: %s, got: %s", td.domain, domain)
		}
		if named != td.name {
			t.Errorf("parse image failed, expected named: %s, got: %s", td.name, named)
		}
	}
}
