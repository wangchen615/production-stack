package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	lorav1alpha1 "github.com/vllm-project/vllm/src/lora-controller/api/v1alpha1"
	"github.com/vllm-project/vllm/src/lora-controller/pkg/placement"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = lorav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create a manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create the placement algorithm
	algorithm := placement.NewDefaultAlgorithm(k8sManager.GetClient(), "default")

	// Setup the controller
	err = (&LoraAdapterReconciler{
		Client:    k8sManager.GetClient(),
		Scheme:    k8sManager.GetScheme(),
		Algorithm: algorithm,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager
	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("LoraAdapter Controller", func() {
	const timeout = time.Second * 10
	const interval = time.Millisecond * 250
	const namespace = "default"

	Context("When creating a LoraAdapter", func() {
		It("Should successfully create and delete a LoraAdapter", func() {
			ctx := context.Background()

			// Create test pods
			pod1 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-1",
					Namespace: namespace,
					Labels: map[string]string{
						"app": "vllm",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vllm",
							Image: "test-image",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod1)).Should(Succeed())

			// Create the LoraAdapter
			loraAdapter := &lorav1alpha1.LoraAdapter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lora",
					Namespace: namespace,
				},
				Spec: lorav1alpha1.LoraAdapterSpec{
					AdapterSource: lorav1alpha1.AdapterSource{
						Type:       "local",
						Repository: "test-repo",
						ModelName:  "test-model",
						ModelPath:  "/path/to/model",
					},
					DeploymentConfig: lorav1alpha1.DeploymentConfig{
						Replicas: 1,
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "vllm",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, loraAdapter)).Should(Succeed())

			// Verify the LoraAdapter was created
			createdLoraAdapter := &lorav1alpha1.LoraAdapter{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-lora",
					Namespace: namespace,
				}, createdLoraAdapter)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify the LoraAdapter spec
			Expect(createdLoraAdapter.Spec.ModelName).Should(Equal("test-model"))
			Expect(createdLoraAdapter.Spec.ModelPath).Should(Equal("/path/to/model"))
			Expect(createdLoraAdapter.Spec.Replicas).Should(Equal(int32(1)))

			// Verify status is updated
			Eventually(func() []string {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-lora",
					Namespace: namespace,
				}, createdLoraAdapter)
				if err != nil {
					return nil
				}
				return createdLoraAdapter.Status.LoadedPods
			}, timeout, interval).Should(ContainElement("test-pod-1"))

			// Cleanup
			Expect(k8sClient.Delete(ctx, loraAdapter)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, pod1)).Should(Succeed())
		})

		It("Should handle invalid configurations", func() {
			ctx := context.Background()

			// Create LoraAdapter with invalid replica count
			invalidLoraAdapter := &lorav1alpha1.LoraAdapter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-lora",
					Namespace: namespace,
				},
				Spec: lorav1alpha1.LoraAdapterSpec{
					ModelName: "test-model",
					ModelPath: "/path/to/model",
					Replicas: -1, // Invalid value
				},
			}
			Expect(k8sClient.Create(ctx, invalidLoraAdapter)).ShouldNot(Succeed())
		})
	})
}) 