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

package conf

import (
	"context"
	"log"

	"github.com/ketches/registry-proxy/pkg/global"
	"github.com/ketches/registry-proxy/pkg/kube"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func HotLoading() {
	configMapInformerFactory := informerscorev1.NewFilteredConfigMapInformer(kube.Client(), global.TargetNamespace, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, func(options *metav1.ListOptions) {
		options.FieldSelector = "metadata.name=" + global.TargetConfigMap
	})
	_, err := kube.Client().CoreV1().ConfigMaps(global.TargetNamespace).Get(context.Background(), global.TargetConfigMap, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		// ConfigMap not exists, create if from defaultConfig
		out, err := yaml.Marshal(defaultConfig)
		if err != nil {
			log.Printf("marshal config failed: %v", err)
		} else {
			_, err := kube.Client().CoreV1().ConfigMaps(global.TargetNamespace).Create(context.Background(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      global.TargetConfigMap,
					Namespace: global.TargetNamespace,
				},
				Data: map[string]string{
					global.TargetConfigMapPath: string(out),
				},
			}, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Create configmap failed: %v", err)
			} else {
				log.Printf("Create configmap %s/%s success", global.TargetNamespace, global.TargetConfigMap)
			}
		}
	}

	configMapInformerFactory.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*corev1.ConfigMap)
			if ok {
				log.Printf("ConfigMap %s/%s added", cm.Namespace, cm.Name)
				tryResetConfigFromConfigMap(obj.(*corev1.ConfigMap))
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCM, ok1 := oldObj.(*corev1.ConfigMap)
			newCM, ok2 := newObj.(*corev1.ConfigMap)
			if ok1 && ok2 && oldCM.ResourceVersion != newCM.ResourceVersion {
				log.Printf("ConfigMap %s/%s updated", newCM.Namespace, newCM.Name)
				tryResetConfigFromConfigMap(newCM)
			}
		},
		DeleteFunc: func(obj interface{}) {
			cm, ok := obj.(*corev1.ConfigMap)
			if ok {
				log.Printf("ConfigMap %s/%s deleted", cm.Namespace, cm.Name)
				tryResetConfigFromConfigMap(nil)
			}
		},
	})
	go func() {
		configMapInformerFactory.Run(wait.NeverStop)
	}()
	go func() {
		if !cache.WaitForCacheSync(wait.NeverStop, configMapInformerFactory.HasSynced) {
			panic("timed out waiting for caches to sync")
		}
	}()
}

type configModel struct {
	ExcludeNamespaces []string `yaml:"excludeNamespaces"`
	IncludeNamespaces []string `yaml:"includeNamespaces"`
	IncludeRegistries []string `yaml:"includeRegistries"`
}

var defaultConfig = configModel{
	ExcludeNamespaces: []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
	},
	IncludeNamespaces: []string{
		"*",
	},
	IncludeRegistries: maps.Keys(global.SupportedRegistries),
}

var config = defaultConfig

func printCurrentConfig(c configModel) {
	out, err := yaml.Marshal(c)
	if err != nil {
		log.Printf("marshal config failed: %v", err)
		return
	}
	log.Printf("Current registry proxy config: \n%s", string(out))
}

func GetExcludeNamespaces() []string {
	return config.ExcludeNamespaces
}

func GetIncludeNamespaces() []string {
	return config.IncludeNamespaces
}

func GetIncludeRegistries() []string {
	return config.IncludeRegistries
}

func tryResetConfigFromConfigMap(cm *corev1.ConfigMap) {
	if cm != nil {
		err := yaml.Unmarshal([]byte(cm.Data[global.TargetConfigMapPath]), &config)
		if err != nil {
			log.Printf("reset config failed: %v", err)
		}
	} else {
		// ConfigMap is deleted, reset to default
		config = defaultConfig
	}
	printCurrentConfig(config)
}
