/*
Copyright 2021.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MyOwnShellSpec defines the desired state of MyOwnShell
type MyOwnShellSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MyOwnShell. Edit myownshell_types.go to remove/update
	Size int32 `json:"size"`
}

// MyOwnShellStatus defines the observed state of MyOwnShell
type MyOwnShellStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyOwnShell is the Schema for the myownshells API
type MyOwnShell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyOwnShellSpec   `json:"spec,omitempty"`
	Status MyOwnShellStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyOwnShellList contains a list of MyOwnShell
type MyOwnShellList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyOwnShell `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyOwnShell{}, &MyOwnShellList{})
}
