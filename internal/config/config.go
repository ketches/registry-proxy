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

package config

import (
	"log"

	"github.com/ketches/registry-proxy/pkg/util"
	"k8s.io/apimachinery/pkg/labels"
)

// config is the registry proxy config
type config struct {
	// Enabled is the flag to enable the registry proxy
	Enabled bool `yaml:"enabled"`
	// Proxies is the map of registry domain and proxy domain
	Proxies map[string]string `yaml:"proxies"`
	// ExcludeNamespaces is the list of namespaces that will not be proxied
	ExcludeNamespaces []string `yaml:"excludeNamespaces"`
	// IncludeNamespaces is the list of namespaces that will be proxied
	IncludeNamespaces []string `yaml:"includeNamespaces"`
	// PodSelector is the selector to select the pods that will be proxied
	PodSelector labels.Set `yaml:"podSelector"`
	// NamespaceSelector is the selector to select the namespaces that will be proxied
	NamespaceSelector labels.Set `yaml:"namespaceSelector"`
}

var defaultProxies = map[string]string{
	"docker.io":       "docker.m.daocloud.io",
	"registry.k8s.io": "k8s.m.daocloud.io",
	"quay.io":         "quay.m.daocloud.io",
	"ghcr.io":         "ghcr.m.daocloud.io",
	"gcr.io":          "gcr.m.daocloud.io",
	"k8s.gcr.io":      "k8s-gcr.m.daocloud.io",
}

var defaultConfig = config{
	Enabled: true,
	Proxies: defaultProxies,
	ExcludeNamespaces: []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"registry-proxy",
	},
	IncludeNamespaces: []string{
		"*",
	},
	PodSelector:       labels.Set{},
	NamespaceSelector: labels.Set{},
}

// configInstance is the singleton config instance
var configInstance = &defaultConfig

// Get get the singleton config instance
func Get() *config {
	return configInstance
}

// GetProxies get the singleton config instance's proxies
func GetProxies() map[string]string {
	return configInstance.Proxies
}

// GetExcludeNamespaces get the singleton config instance's excludeNamespaces
func GetExcludeNamespaces() []string {
	return configInstance.ExcludeNamespaces
}

// GetIncludeNamespaces get the singleton config instance's includeNamespaces
func GetIncludeNamespaces() []string {
	return configInstance.IncludeNamespaces
}

// Enabled get the singleton config instance's enabled
func Enabled() bool {
	return configInstance.Enabled
}

// PodSelector get the singleton config instance's pod selector
func PodSelector() labels.Set {
	return configInstance.PodSelector
}

// NamespaceSelector get the singleton config instance's namespace selector
func NamespaceSelector() labels.Set {
	return configInstance.NamespaceSelector
}

// Reset reset the singleton config instance.
// If in is empty, reset to default config
func Reset(in []byte) {
	if len(in) > 0 {
		configInstance = &config{} // reset to empty config
		err := util.UnmarshalYAML(in, configInstance)
		if err != nil {
			log.Printf("Reset config failed: %v", err)
		}
	} else {
		// reset to default config
		configInstance = &defaultConfig
	}
	printCurrentConfig()
}

// printCurrentConfig print current config
func printCurrentConfig() {
	out, err := util.MarshalYAML(configInstance)
	if err != nil {
		log.Printf("Marshal config failed: %v", err)
		return
	}
	log.Printf("Current registry proxy config: \n%s", string(out))
}
