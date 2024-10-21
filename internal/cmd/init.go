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
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/ketches/registry-proxy/internal/config"
	"github.com/ketches/registry-proxy/internal/global"
	"github.com/ketches/registry-proxy/pkg/kube"
	"github.com/ketches/registry-proxy/pkg/util"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	certuitl "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/retry"
)

var dns = fmt.Sprintf("%s.%s.svc", global.TargetName, global.TargetNamespace)

var (
	cert []byte
	key  []byte
)

// Init initializes the registry-proxy. Do the following things:
//
// 1. Watch the ConfigMap and trigger config reset.
//
// 2. Create or reset the TLS cert and key secret.
//
// 3. Create or reset the MutatingWebhookConfiguration to use the new cert and config.
func Init() {
	fmt.Println("Welcome to use registry-proxy!")

	runConfigMapInformer()

	applyTLSCertSecret()

	applyWebhook()
}

// configMapEventHandler handles the ConfigMap events.
var configMapEventHandler = cache.ResourceEventHandlerFuncs{
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
}

// runConfigMapInformer watches the ConfigMap and triggers config reset.
func runConfigMapInformer() {
	configMapInformerFactory := informerscorev1.NewFilteredConfigMapInformer(kube.Client(), global.TargetNamespace, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, func(options *metav1.ListOptions) {
		options.FieldSelector = "metadata.name=" + global.ConfigMapName
	})
	if _, err := kube.Client().CoreV1().ConfigMaps(global.TargetNamespace).Get(context.Background(), global.ConfigMapName, metav1.GetOptions{}); err != nil && errors.IsNotFound(err) {
		// ConfigMap not exists, create if from defaultConfig
		out, err := util.MarshalYAML(config.Get())
		if err != nil {
			log.Printf("Marshal config failed: %v", err)
		} else {
			_, err := kube.Client().CoreV1().ConfigMaps(global.TargetNamespace).Create(context.Background(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      global.ConfigMapName,
					Namespace: global.TargetNamespace,
				},
				Data: map[string]string{
					global.ConfigMapPath: string(out),
				},
			}, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Create configmap failed: %v", err)
			} else {
				log.Printf("Create configmap %s/%s success", global.TargetNamespace, global.ConfigMapName)
			}
		}
	}

	configMapInformerFactory.AddEventHandler(configMapEventHandler)
	go func() {
		configMapInformerFactory.Run(wait.NeverStop)
	}()
	go func() {
		if !cache.WaitForCacheSync(wait.NeverStop, configMapInformerFactory.HasSynced) {
			panic("timed out waiting for caches to sync")
		}
	}()
}

// tryResetConfigFromConfigMap tries to reset the config from the ConfigMap.
func tryResetConfigFromConfigMap(cm *corev1.ConfigMap) {
	var data []byte
	if cm != nil {
		data = []byte(cm.Data[global.ConfigMapPath])
	}

	// reset config
	config.Reset(data)

	// try to update the MutatingWebhookConfiguration
	applyWebhook()
}

// constructWebhook constructs a MutatingWebhookConfiguration.
func constructWebhook() *admissionregistrationv1.MutatingWebhookConfiguration {
	result := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: global.WebhookName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: cert,
					Service: &admissionregistrationv1.ServiceReference{
						Name:      global.TargetName,
						Namespace: global.TargetNamespace,
						Path:      util.Ptr(global.WebhookServicePath),
					},
				},
				FailurePolicy:     util.Ptr(admissionregistrationv1.Fail),
				MatchPolicy:       util.Ptr(admissionregistrationv1.Exact),
				Name:              dns,
				NamespaceSelector: &metav1.LabelSelector{},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       util.Ptr(admissionregistrationv1.NamespacedScope),
						},
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
					},
				},
				SideEffects:    util.Ptr(admissionregistrationv1.SideEffectClassNone),
				TimeoutSeconds: util.Ptr(int32(5)),
			},
		},
	}

	if v := config.GetExcludeNamespaces(); len(v) > 0 {
		var selector metav1.LabelSelectorRequirement
		if slices.Contains(v, "*") {
			// exclude all namespaces
			selector = metav1.LabelSelectorRequirement{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpDoesNotExist, // match none
			}
		} else {
			selector = metav1.LabelSelectorRequirement{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpNotIn,
				Values:   v,
			}
		}
		result.Webhooks[0].NamespaceSelector.MatchExpressions = append(result.Webhooks[0].NamespaceSelector.MatchExpressions, selector)
	}

	if v := config.GetIncludeNamespaces(); len(v) > 0 {
		var selector metav1.LabelSelectorRequirement
		if slices.Contains(v, "*") {
			// include all namespaces
			selector = metav1.LabelSelectorRequirement{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpExists, // match all
			}
		} else {
			selector = metav1.LabelSelectorRequirement{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpIn,
				Values:   v,
			}
		}
		result.Webhooks[0].NamespaceSelector.MatchExpressions = append(result.Webhooks[0].NamespaceSelector.MatchExpressions, selector)
	}

	// Set namespace selector from config
	if selector := config.NamespaceSelector(); len(selector) > 0 {
		for k, v := range selector {
			result.Webhooks[0].NamespaceSelector.MatchExpressions = append(result.Webhooks[0].NamespaceSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      k,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{v},
			})
		}
	}

	// set owner reference, so that the MutatingWebhookConfiguration will be deleted on uninstall
	ns, _ := kube.Client().CoreV1().Namespaces().Get(context.Background(), global.TargetNamespace, metav1.GetOptions{})
	if ns != nil {
		result.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(ns, corev1.SchemeGroupVersion.WithKind("Namespace")),
		})
	}

	return result
}

// applyTLSCertSecret applies the TLS cert and key secret.
func applyTLSCertSecret() {
	if secret, err := kube.Client().CoreV1().Secrets(global.TargetNamespace).Get(context.Background(), global.WebhookTLSCertSecretName, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			cert, key, err = certuitl.GenerateSelfSignedCertKey(dns, nil, []string{dns})
			if err != nil {
				log.Fatalf("Generate self-signed cert and key failed: %v", err)
			}
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      global.WebhookTLSCertSecretName,
					Namespace: global.TargetNamespace,
				},
				Data: map[string][]byte{
					corev1.TLSCertKey:       cert,
					corev1.TLSPrivateKeyKey: key,
				},
				Type: corev1.SecretTypeTLS,
			}
			if _, err := kube.Client().CoreV1().Secrets(global.TargetNamespace).Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Fatalf("Create secret failed: %v", err)
				}
			}
		} else {
			log.Fatalf("Get secret failed: %v", err)
		}
	} else {
		cert = secret.Data[corev1.TLSCertKey]
		key = secret.Data[corev1.TLSPrivateKeyKey]
	}

	// create the cert and key files
	err := os.WriteFile(global.WebhookServiceTLSCertFile, cert, 0644)
	if err != nil {
		log.Fatalf("Write cert failed: %v", err)
	}
	err = os.WriteFile(global.WebhookServiceTLSKeyFile, key, 0644)
	if err != nil {
		log.Fatalf("Write key failed: %v", err)
	}
}

// applyWebhook applies the MutatingWebhookConfiguration.
func applyWebhook() {
	if !config.Enabled() {
		err := kube.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), global.WebhookName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Fatalf("Delete MutatingWebhookConfigurations failed: %v", err)
		}
		return
	}

	if mwc, err := kube.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), global.WebhookName, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			mwc = constructWebhook()
			if _, err := kube.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), mwc, metav1.CreateOptions{}); err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Fatalf("Create MutatingWebhookConfigurations failed: %v", err)
				}
			}
		} else {
			log.Fatalf("Get MutatingWebhookConfigurations failed: %v", err)
		}
	} else {
		new := constructWebhook()
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			new.ResourceVersion = mwc.ResourceVersion
			_, err := kube.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), new, metav1.UpdateOptions{})
			if err != nil {
				mwc, err = kube.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), global.WebhookName, metav1.GetOptions{})
			}
			return err
		})
		if err != nil {
			log.Fatalf("Update MutatingWebhookConfigurations failed: %v", err)
		}
	}
}
