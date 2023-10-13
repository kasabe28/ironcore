// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package annotations

import (
	"time"

	"github.com/onmetal/controller-utils/metautils"
	commonv1beta1 "github.com/onmetal/onmetal-api/api/common/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HasReconcileAnnotation(o metav1.Object) bool {
	return metautils.HasAnnotation(o, commonv1beta1.ReconcileRequestAnnotation)
}

func SetReconcileAnnotation(o metav1.Object) {
	metautils.SetAnnotation(o, commonv1beta1.ReconcileRequestAnnotation, time.Now().Format(time.RFC3339Nano))
}

func RemoveReconcileAnnotation(o metav1.Object) {
	metautils.DeleteAnnotation(o, commonv1beta1.ReconcileRequestAnnotation)
}

func IsExternallyManaged(o metav1.Object) bool {
	return metautils.HasAnnotation(o, commonv1beta1.ManagedByAnnotation)
}

func IsEphemeralManagedBy(o metav1.Object, manager string) bool {
	actual, ok := o.GetAnnotations()[commonv1beta1.EphemeralManagedByAnnotation]
	return ok && actual == manager
}

func IsDefaultEphemeralControlledBy(o metav1.Object, owner metav1.Object) bool {
	return metav1.IsControlledBy(o, owner) && IsEphemeralManagedBy(o, commonv1beta1.DefaultEphemeralManager)
}

func SetDefaultEphemeralManagedBy(o metav1.Object) {
	metautils.SetAnnotation(o, commonv1beta1.EphemeralManagedByAnnotation, commonv1beta1.DefaultEphemeralManager)
}

func SetExternallyMangedBy(o metav1.Object, manager string) {
	metautils.SetAnnotation(o, commonv1beta1.ManagedByAnnotation, manager)
}

func RemoveExternallyManaged(o metav1.Object) {
	metautils.DeleteAnnotation(o, commonv1beta1.ManagedByAnnotation)
}
