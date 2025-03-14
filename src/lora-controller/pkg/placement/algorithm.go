package placement

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Algorithm defines the interface for LoRA placement algorithms
type Algorithm interface {
	// PlaceAdapter determines which pods should load a given LoRA adapter
	PlaceAdapter(ctx context.Context, podSelector labels.Selector) ([]corev1.Pod, error)
}

// DefaultAlgorithm implements the default placement strategy
type DefaultAlgorithm struct {
	client    client.Client
	namespace string
}

// NewDefaultAlgorithm creates a new instance of DefaultAlgorithm
func NewDefaultAlgorithm(client client.Client, namespace string) *DefaultAlgorithm {
	return &DefaultAlgorithm{
		client:    client,
		namespace: namespace,
	}
}

// PlaceAdapter implements the default strategy of placing adapters on all matching pods
func (d *DefaultAlgorithm) PlaceAdapter(ctx context.Context, podSelector labels.Selector) ([]corev1.Pod, error) {
	var podList corev1.PodList
	if err := d.client.List(ctx, &podList, &client.ListOptions{
		Namespace:     d.namespace,
		LabelSelector: podSelector,
	}); err != nil {
		return nil, err
	}

	return podList.Items, nil
} 