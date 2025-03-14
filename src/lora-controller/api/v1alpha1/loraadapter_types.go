package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

// AdapterSource defines the source of the LoRA adapter
type AdapterSource struct {
	// Type is the type of adapter source (e.g., "local", "s3", "http")
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Repository is the repository where the adapter is stored
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// ModelPath is the path to the LoRA adapter weights
	// +kubebuilder:validation:Required
	ModelPath string `json:"modelPath"`

	// ModelName is the name of the model to apply the LoRA adapter to
	// +kubebuilder:validation:Required
	ModelName string `json:"modelName"`
}

// DeploymentConfig defines how the adapter should be deployed
type DeploymentConfig struct {
	// Replicas is the number of replicas that should load this adapter
	// +kubebuilder:validation:Minimum=0
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// PodSelector selects the pods that should load this adapter
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
}

// LoraAdapterSpec defines the desired state of LoraAdapter
type LoraAdapterSpec struct {
	// AdapterSource defines where to get the LoRA adapter from
	// +kubebuilder:validation:Required
	AdapterSource AdapterSource `json:"adapterSource"`

	// DeploymentConfig defines how the adapter should be deployed
	// +optional
	DeploymentConfig DeploymentConfig `json:"deploymentConfig,omitempty"`
}

// PodAssignment represents a pod that has been assigned to load this adapter
type PodAssignment struct {
	// Pod represents the pod information
	Pod corev1.ObjectReference `json:"pod"`

	// Status represents the current status of the assignment
	Status string `json:"status"`

	// Adapters is the list of adapters assigned to this pod
	Adapters []string `json:"adapters"`
}

// LoadedAdapter represents an adapter that has been loaded into a pod
type LoadedAdapter struct {
	// Name is the name of the adapter
	Name string `json:"name"`

	// Path is the path where the adapter is loaded
	Path string `json:"path"`

	// Status represents the current status of the loaded adapter
	Status string `json:"status"`

	// LoadTime is when the adapter was loaded
	LoadTime *metav1.Time `json:"loadTime,omitempty"`
}

// LoraAdapterStatus defines the observed state of LoraAdapter
type LoraAdapterStatus struct {
	// Phase represents the current phase of the adapter deployment
	// +optional
	Phase string `json:"phase,omitempty"`

	// LoadedPods tracks which pods have loaded this adapter
	// +optional
	LoadedPods []string `json:"loadedPods,omitempty"`

	// PodAssignments tracks the assignment status of pods
	// +optional
	PodAssignments []PodAssignment `json:"podAssignments,omitempty"`

	// LoadedAdapters tracks the loading status of adapters
	// +optional
	LoadedAdapters []LoadedAdapter `json:"loadedAdapters,omitempty"`

	// Conditions represent the latest available observations of the adapter's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LoraAdapter is the Schema for the loraadapters API
type LoraAdapter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoraAdapterSpec   `json:"spec,omitempty"`
	Status LoraAdapterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LoraAdapterList contains a list of LoraAdapter
type LoraAdapterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoraAdapter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoraAdapter{}, &LoraAdapterList{})
} 