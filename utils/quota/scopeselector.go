// Copyright 2023 OnMetal authors
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

package quota

import (
	corev1beta1 "github.com/onmetal/onmetal-api/api/core/v1beta1"
)

func GetResourceScopeSelectorRequirements(scopeSelector *corev1beta1.ResourceScopeSelector) []corev1beta1.ResourceScopeSelectorRequirement {
	if scopeSelector == nil {
		return nil
	}

	return scopeSelector.MatchExpressions
}
