package controllers

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lorav1alpha1 "github.com/vllm-project/vllm/src/lora-controller/api/v1alpha1"
	"github.com/vllm-project/vllm/src/lora-controller/pkg/placement"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// LoraAdapterReconciler reconciles a LoraAdapter object
type LoraAdapterReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Algorithm placement.Algorithm
}

//+kubebuilder:rbac:groups=production-stack.vllm-project,resources=loraadapters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=production-stack.vllm-project,resources=loraadapters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=production-stack.vllm-project,resources=loraadapters/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *LoraAdapterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LoraAdapter instance
	var loraAdapter lorav1alpha1.LoraAdapter
	if err := r.Get(ctx, req.NamespacedName, &loraAdapter); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle different source types
	var adapters []string
	var err error

	switch loraAdapter.Spec.AdapterSource.Type {
	case "huggingface":
		adapters = []string{loraAdapter.Spec.AdapterSource.Repository}
	case "s3", "cos":
		adapters, err = r.discoverAdapters(ctx, &loraAdapter)
	case "local":
		adapters = []string{loraAdapter.Spec.AdapterSource.Repository}
	default:
		err = fmt.Errorf("unsupported adapter source type: %s", loraAdapter.Spec.AdapterSource.Type)
	}

	if err != nil {
		log.Error(err, "Failed to get adapters")
		return ctrl.Result{}, err
	}

	// Parse pod selector
	selector, err := metav1.LabelSelectorAsSelector(&loraAdapter.Spec.DeploymentConfig.PodSelector)
	if err != nil {
		log.Error(err, "Failed to parse pod selector")
		return ctrl.Result{}, err
	}

	// Use placement algorithm to determine pod assignments
	pods, err := r.Algorithm.PlaceAdapter(ctx, selector)
	if err != nil {
		log.Error(err, "Failed to determine pod assignments")
		return ctrl.Result{}, err
	}

	// Update status with pod assignments and loaded adapters
	now := metav1.Now()
	podAssignments := make([]lorav1alpha1.PodAssignment, 0, len(pods))
	loadedAdapters := make([]lorav1alpha1.LoadedAdapter, 0, len(adapters))

	for _, pod := range pods {
		podAssignments = append(podAssignments, lorav1alpha1.PodAssignment{
			Pod:      pod.Name,
			Adapters: adapters,
		})
	}

	for _, adapter := range adapters {
		loadedAdapters = append(loadedAdapters, lorav1alpha1.LoadedAdapter{
			Name:     adapter,
			Path:     adapter, // For s3/cos, this will be the full path
			LoadTime: now,
		})
	}

	// Update status
	loraAdapter.Status.Phase = "Ready"
	loraAdapter.Status.PodAssignments = podAssignments
	loraAdapter.Status.LoadedAdapters = loadedAdapters
	loraAdapter.Status.LastDiscoveryTime = &now

	if err := r.Status().Update(ctx, &loraAdapter); err != nil {
		log.Error(err, "Failed to update LoraAdapter status")
		return ctrl.Result{}, err
	}

	// Requeue for s3/cos sources to periodically check for new adapters
	if loraAdapter.Spec.AdapterSource.Type == "s3" || loraAdapter.Spec.AdapterSource.Type == "cos" {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	return ctrl.Result{}, nil
}

// discoverAdapters finds all adapters matching the pattern in the s3/cos path
func (r *LoraAdapterReconciler) discoverAdapters(ctx context.Context, loraAdapter *lorav1alpha1.LoraAdapter) ([]string, error) {
	// Get storage credentials if specified
	var creds map[string][]byte
	if loraAdapter.Spec.CredentialsSecretRef != nil {
		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{
			Name:      loraAdapter.Spec.CredentialsSecretRef.Name,
			Namespace: loraAdapter.Namespace,
		}, &secret); err != nil {
			return nil, fmt.Errorf("failed to get credentials: %w", err)
		}
		creds = secret.Data
	}

	// Initialize storage client based on type
	var storageClient interface{} // Replace with actual S3/COS client type
	var err error
	if loraAdapter.Spec.AdapterSource.Type == "s3" {
		storageClient, err = r.initS3Client(creds)
	} else {
		storageClient, err = r.initCOSClient(creds)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage client: %w", err)
	}

	// List objects in the path
	objects, err := r.listStorageObjects(ctx, storageClient, loraAdapter.Spec.AdapterSource.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage objects: %w", err)
	}

	// Filter objects based on pattern if specified
	var adapters []string
	if pattern := loraAdapter.Spec.AdapterSource.Pattern; pattern != "" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		for _, obj := range objects {
			if re.MatchString(obj) {
				adapters = append(adapters, obj)
			}
		}
	} else {
		adapters = objects
	}

	// Limit number of adapters if specified
	if max := loraAdapter.Spec.AdapterSource.MaxAdapters; max != nil && int32(len(adapters)) > *max {
		adapters = adapters[:*max]
	}

	return adapters, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LoraAdapterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lorav1alpha1.LoraAdapter{}).
		Complete(r)
}

// Note: Implementation of initS3Client, initCOSClient, and listStorageObjects would go here
// These would handle the actual S3/COS interactions 