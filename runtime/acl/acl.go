/*
Copyright 2021 The Flux authors

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

package acl

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aclapi "github.com/fluxcd/pkg/apis/acl"
)

// Authorization is an ACL helper for asserting access to cross-namespace references.
type Authorization struct {
	client client.Client
}

// NewAuthorization takes a controller runtime client and returns an Authorization object that allows asserting
// access to cross-namespace references.
func NewAuthorization(kubeClient client.Client) *Authorization {
	return &Authorization{client: kubeClient}
}

// HasAccessToRef asserts if a namespaced object has access to a cross-namespace reference based on the ACL defined on the referenced object.
func (a *Authorization) HasAccessToRef(ctx context.Context, object client.Object, reference types.NamespacedName, acl *aclapi.AccessFrom) (bool, error) {
	// grant access if the object is in the same namespace as the reference
	if reference.Namespace == "" || object.GetNamespace() == reference.Namespace {
		return true, nil
	}

	// deny access if no ACL is defined on the reference
	if acl == nil {
		return false, fmt.Errorf("'%s/%s' can't be accessed due to missing ACL labels on 'accessFrom'",
			reference.Namespace, reference.Name)
	}

	// get the object's namespace labels
	var sourceNamespace corev1.Namespace
	if err := a.client.Get(ctx, types.NamespacedName{Name: object.GetNamespace()}, &sourceNamespace); err != nil {
		return false, err
	}
	sourceLabels := sourceNamespace.GetLabels()

	// check if the object's namespace labels match any ACL
	for _, selector := range acl.NamespaceSelectors {
		sel, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: selector.MatchLabels})
		if err != nil {
			return false, err
		}
		if sel.Matches(labels.Set(sourceLabels)) {
			return true, nil
		}
	}

	return false, fmt.Errorf("'%s/%s' can't be accessed due to ACL labels mismatch on namespace '%s'",
		reference.Namespace, reference.Name, object.GetNamespace())
}
