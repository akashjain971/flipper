package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Flipper struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FlipperSpec `json:"spec,omitempty"`
}

type FlipperSpec struct {
	ReconcileFrequency string       `json:"reconcileFrequency,omitempty"`
	Interval           string       `json:"interval,omitempty"`
	Match              FlipperMatch `json:"match,omitempty"`
}

type FlipperMatch struct {
	Labels     map[string]string `json:"labels,omitempty"`
	Namespaces []string          `json:"namespaces,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FlipperList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Flipper `json:"items"`
}
