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

	"gopkg.in/yaml.v3"
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
}

var defaultProxies = map[string]string{
	"docker.io":            "docker.ketches.cn",
	"registry.k8s.io":      "k8s.ketches.cn",
	"quay.io":              "quay.ketches.cn",
	"ghcr.io":              "ghcr.ketches.cn",
	"gcr.io":               "gcr.ketches.cn",
	"k8s.gcr.io":           "k8s-gcr.ketches.cn",
	"docker.cloudsmith.io": "cloudsmith.ketches.cn",
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

// Reset reset the singleton config instance.
// If in is empty, reset to default config
func Reset(in []byte) {
	if len(in) > 0 {
		err := yaml.Unmarshal([]byte(in), configInstance)
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
	out, err := yaml.Marshal(configInstance)
	if err != nil {
		log.Printf("Marshal config failed: %v", err)
		return
	}
	log.Printf("Current registry proxy config: \n%s", string(out))
}
