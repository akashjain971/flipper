package controllers

import (
	"github.com/akashjain971/flipper/api/flipper.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TestReconcilerFrequency = "2s"
	TestFlipperInterval     = "5s"
)

const (
	TestDefaultNamespace = "default"
	TestFlipperName      = "service-mesh-flipper"
	TestDeployName       = "nginx"
	TestPodName          = "nginx"
	TestDeployImage      = "nginx"
)

var (
	TestFlipperLabels     = map[string]string{"mesh": "true"}
	TestFlipperNamespaces = []string{TestDefaultNamespace}

	TestPodLabels            = map[string]string{"app": TestPodName}
	TestDeployNamespacedName = types.NamespacedName{Namespace: TestDefaultNamespace, Name: TestDeployName}
)

func getDeployment(deployLabels, podAnnotations map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestDeployName,
			Namespace: TestDefaultNamespace,
			Labels:    deployLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: TestPodLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      TestPodLabels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: TestPodName, Image: TestDeployImage}},
				},
			},
		},
	}
}

func getFlipper(freq, interval string, labels map[string]string, namespaces []string) *v1alpha1.Flipper {
	return &v1alpha1.Flipper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestFlipperName,
			Namespace: TestDefaultNamespace,
		},
		Spec: v1alpha1.FlipperSpec{
			ReconcileFrequency: freq,
			Interval:           interval,
			Match: v1alpha1.FlipperMatch{
				Labels:     labels,
				Namespaces: namespaces,
			},
		},
	}
}
