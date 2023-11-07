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

import (
	"testing"
)

func TestParseImage(t *testing.T) {
	testdata := []struct {
		image  string
		domain string
		name   string
	}{
		{
			image:  "username/image:tag",
			domain: "docker.io",
			name:   "username/image:tag",
		}, {
			image:  "image:tag",
			domain: "docker.io",
			name:   "library/image:tag",
		}, {
			image:  "gcr.io/username/image:tag",
			domain: "gcr.io",
			name:   "username/image:tag",
		}, {
			image:  "ghcr.io/username/image:tag",
			domain: "ghcr.io",
			name:   "username/image:tag",
		}, {
			image:  "ghcr.io/username/namespace/image:tag",
			domain: "ghcr.io",
			name:   "username/namespace/image:tag",
		}, {
			image:  "k8s.gcr.io/image:tag",
			domain: "k8s.gcr.io",
			name:   "image:tag",
		}, {
			image:  "k8s.gcr.io/username/image:tag",
			domain: "k8s.gcr.io",
			name:   "username/image:tag",
		}, {
			image:  "registry.k8s.io/image:tag",
			domain: "registry.k8s.io",
			name:   "image:tag",
		}, {
			image:  "registry.k8s.io/username/image:tag",
			domain: "registry.k8s.io",
			name:   "username/image:tag",
		}, {
			image:  "quay.io/username/image:tag",
			domain: "quay.io",
			name:   "username/image:tag",
		},
	}

	for _, td := range testdata {
		domain, named, err := Parse(td.image)
		if err != nil {
			t.Errorf("parse image failed: %v", err)
		}
		if domain != td.domain {
			t.Errorf("parse image %s failed, expected domain: %s, got: %s", td.image, td.domain, domain)
		}
		if named != td.name {
			t.Errorf("parse image %s failed, expected named: %s, got: %s", td.image, td.name, named)
		}
	}
}
