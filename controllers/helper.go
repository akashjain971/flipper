package controllers

import (
	"time"

	"github.com/akashjain971/flipper/api/flipper.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

const RestartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"
const DefaultInterval = "10m"
const DefaultReconcileFrequency = "1m"

var DefaultNamespaces = []string{""}

func setFlipperDefaults(spec *v1alpha1.FlipperSpec) {
	if spec.ReconcileFrequency == "" {
		spec.ReconcileFrequency = DefaultReconcileFrequency
	}

	if spec.Interval == "" {
		spec.Interval = DefaultInterval
	}

	if len(spec.Match.Namespaces) == 0 {
		spec.Match.Namespaces = DefaultNamespaces
	}
}

func getRestartedAtTime(deployment appsv1.Deployment) (time.Time, error) {
	restartedAt := deployment.Spec.Template.Annotations[RestartedAtAnnotation]

	if restartedAt == "" {
		// restartedAt annotation not set for new deployment, fallback to creation timestamp
		return deployment.CreationTimestamp.Time, nil
	}

	// use restartedAt annotation next time onwards
	return time.Parse(time.RFC3339, restartedAt)
}

func isRestartDue(t time.Time, d time.Duration) bool {
	return time.Now().UTC().Sub(t) > d
}
