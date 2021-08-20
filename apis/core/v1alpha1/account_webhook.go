/*
 * Copyright (c) 2021 by the OnMetal authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var accountlog = logf.Log.WithName("account-resource")

//+kubebuilder:webhook:path=/mutate-core-onmetal-de-v1alpha1-account,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.onmetal.de,resources=accounts,verbs=create;update,versions=v1alpha1,name=maccount.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Account{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Account) Default() {
	accountlog.Info("default", "name", r.Name)
	// Defaulting code goes here
}

//+kubebuilder:webhook:path=/validate-core-onmetal-de-v1alpha1-account,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.onmetal.de,resources=accounts,verbs=create;update;delete,versions=v1alpha1,name=vaccount.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Account{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Account) ValidateCreate() error {
	accountlog.Info("validate create", "name", r.Name)
	return r.validateAccount()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Account) ValidateUpdate(old runtime.Object) error {
	accountlog.Info("validate update", "name", r.Name)
	return r.validateAccountUpdate(old)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Account) ValidateDelete() error {
	accountlog.Info("validate delete", "name", r.Name)
	return r.validateAccountDelete()
}
