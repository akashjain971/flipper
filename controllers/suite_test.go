/*
Copyright 2022.

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

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/akashjain971/flipper/api/flipper.io/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

var ctx context.Context
var cancel context.CancelFunc

var deployment *appsv1.Deployment = getDeployment(nil, nil)
var flipper *v1alpha1.Flipper = getFlipper("", "", nil, nil)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	Expect(os.Setenv("KUBEBUILDER_ASSETS", "../testbin/bin/")).To(Succeed())
	// Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
	Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	controller := &FlipperReconciler{
		Client:    k8sClient,
		Scheme:    scheme.Scheme,
		Clientset: kubernetes.NewForConfigOrDie(cfg),
	}

	err = controller.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

	go func() {
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to start manager")
	}()

	reconcileFreq, err := time.ParseDuration(TestReconcilerFrequency)
	Expect(err).NotTo(HaveOccurred(), "failed to parse test reconcile frequency")

	interval, err := time.ParseDuration(TestFlipperInterval)
	Expect(err).NotTo(HaveOccurred(), "failed to parse test interval")

	SetDefaultEventuallyTimeout(interval * 2)
	SetDefaultEventuallyPollingInterval(reconcileFreq)
	SetDefaultConsistentlyDuration(interval * 2)
	SetDefaultConsistentlyPollingInterval(reconcileFreq)

	err = k8sClient.Create(ctx, deployment)
	Expect(err).To(Not(HaveOccurred()))

	err = k8sClient.Create(ctx, flipper)
	Expect(err).To(Not(HaveOccurred()))
})

// 1. flipper with invalid interval
// 2. flipper with invalid frequency
// 3. flipper with non matching labels

// 4. flipper with matching labels
// 5. flipper with outdated restarted at annotation
// 6. flipper with future restarted at annotation

var _ = Describe("test flipper", func() {
	Context("flipper with invalid interval", func() {
		It("should set invalid flipper interval", func() {
			deploy := getDeployment(TestFlipperLabels, nil)
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper(TestReconcilerFrequency, "12xx", TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should not restart the deployment pods", func() {
			Consistently(func() string {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				return deploy.Spec.Template.Annotations[RestartedAtAnnotation]
			}).Should(BeEmpty())
		})
	})

	Context("flipper with invalid frequency", func() {
		It("should set invalid flipper frequency", func() {
			deploy := getDeployment(TestFlipperLabels, nil)
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper("10xx", TestFlipperInterval, TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should not restart the deployment pods", func() {
			Consistently(func() string {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				return deploy.Spec.Template.Annotations[RestartedAtAnnotation]
			}).Should(BeEmpty())
		})
	})

	Context("flipper with non matching labels", func() {
		It("should set non matching deployment labels", func() {
			deploy := getDeployment(map[string]string{"mesh": "false"}, nil)
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper(TestReconcilerFrequency, TestFlipperInterval, TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should not restart the deployment pods", func() {
			Consistently(func() string {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				return deploy.Spec.Template.Annotations[RestartedAtAnnotation]
			}).Should(BeEmpty())
		})
	})

	Context("flipper with matching labels", func() {
		It("should set matching deployment labels as flipper", func() {
			deploy := getDeployment(TestFlipperLabels, nil)
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper(TestReconcilerFrequency, TestFlipperInterval, TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should restart the deployment pods", func() {
			Eventually(func() string {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				return deploy.Spec.Template.Annotations[RestartedAtAnnotation]
			}).Should(Not(BeEmpty()))
		})
	})

	Context("deployment with outdated restarted at annotation", func() {
		outdatedTime := time.Now().Add(-1 * time.Hour).UTC()

		It("should set outdated restarted at annotations", func() {
			deploy := getDeployment(TestFlipperLabels, map[string]string{RestartedAtAnnotation: outdatedTime.Format(time.RFC3339)})
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper(TestReconcilerFrequency, TestFlipperInterval, TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should update the restarted at annotation to a time greater than outdated time", func() {
			Eventually(func() time.Time {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				restartedAt, err := time.Parse(time.RFC3339, deploy.Spec.Template.Annotations[RestartedAtAnnotation])
				Expect(err).To(Not(HaveOccurred()))
				return restartedAt
			}).Should(BeTemporally(">", outdatedTime))
		})
	})

	Context("deployment with future restarted at annotation", func() {
		futureTime := time.Now().Add(time.Hour).UTC()

		It("should set future restarted at annotations", func() {
			deploy := getDeployment(TestFlipperLabels, map[string]string{RestartedAtAnnotation: futureTime.Format(time.RFC3339)})
			err := k8sClient.Patch(ctx, deploy, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))

			flipper := getFlipper(TestReconcilerFrequency, TestFlipperInterval, TestFlipperLabels, TestFlipperNamespaces)
			err = k8sClient.Patch(ctx, flipper, client.Merge, &client.PatchOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should not update the restarted at annotation", func() {
			Consistently(func() time.Time {
				deploy := appsv1.Deployment{}
				err := k8sClient.Get(ctx, TestDeployNamespacedName, &deploy)
				Expect(err).To(Not(HaveOccurred()))
				restartedAt, err := time.Parse(time.RFC3339, deploy.Spec.Template.Annotations[RestartedAtAnnotation])
				Expect(err).To(Not(HaveOccurred()))
				return restartedAt
			}).Should(BeTemporally("~", futureTime, time.Second))
		})
	})
})

var _ = AfterSuite(func() {
	err := k8sClient.Delete(ctx, deployment, &client.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Delete(ctx, flipper, &client.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())

	cancel()

	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
