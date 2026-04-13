// Package webhook hosts the validating admission webhooks for the operator
// CRDs. Validation is deliberately structural (shape of user input) — runtime
// failures are the reconciler's job.
package webhook

import (
	"context"

	"github.com/robfig/cron/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	observabilityv1 "github.com/MerlionOS/tsdb-operator/api/v1"
)

// +kubebuilder:webhook:path=/validate-observability-merlionos-org-v1-prometheuscluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.merlionos.org,resources=prometheusclusters,verbs=create;update,versions=v1,name=vprometheuscluster.merlionos.org,admissionReviewVersions=v1

// PrometheusClusterValidator is the admission plugin for PrometheusCluster.
type PrometheusClusterValidator struct{}

// SetupWithManager registers the validator on mgr's webhook server.
func (v *PrometheusClusterValidator) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &observabilityv1.PrometheusCluster{}).
		WithValidator(v).
		Complete()
}

// ValidateCreate is called by the webhook server on create.
func (v *PrometheusClusterValidator) ValidateCreate(_ context.Context, pc *observabilityv1.PrometheusCluster) (admission.Warnings, error) {
	return nil, v.validate(pc)
}

// ValidateUpdate is called by the webhook server on update.
func (v *PrometheusClusterValidator) ValidateUpdate(_ context.Context, _, pc *observabilityv1.PrometheusCluster) (admission.Warnings, error) {
	return nil, v.validate(pc)
}

// ValidateDelete is a no-op; tsdb-operator owns cleanup via finalizers.
func (v *PrometheusClusterValidator) ValidateDelete(_ context.Context, _ *observabilityv1.PrometheusCluster) (admission.Warnings, error) {
	return nil, nil
}

// Validate is exported so unit tests can exercise the rules without a
// running webhook server.
func (v *PrometheusClusterValidator) Validate(pc *observabilityv1.PrometheusCluster) error {
	return v.validate(pc)
}

func (v *PrometheusClusterValidator) validate(pc *observabilityv1.PrometheusCluster) error {
	var errs field.ErrorList
	specPath := field.NewPath("spec")

	if pc.Spec.Replicas < 1 {
		errs = append(errs, field.Invalid(specPath.Child("replicas"), pc.Spec.Replicas,
			"must be >= 1"))
	}

	backupPath := specPath.Child("backup")
	if pc.Spec.Backup.Enabled {
		if pc.Spec.Backup.Bucket == "" {
			errs = append(errs, field.Required(backupPath.Child("bucket"),
				"bucket is required when backup is enabled"))
		}
		if pc.Spec.Backup.Schedule != "" {
			if _, err := cron.ParseStandard(pc.Spec.Backup.Schedule); err != nil {
				errs = append(errs, field.Invalid(backupPath.Child("schedule"),
					pc.Spec.Backup.Schedule, "not a valid cron expression: "+err.Error()))
			}
		}
	}

	for i, rw := range pc.Spec.RemoteWrite {
		if rw.URL == "" {
			errs = append(errs, field.Required(
				specPath.Child("remoteWrite").Index(i).Child("url"),
				"url is required"))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		pc.GroupVersionKind().GroupKind(),
		pc.Name,
		errs,
	)
}

// Register keeps the webhook package's public surface small for cmd/main.go.
func Register(mgr ctrl.Manager) error {
	return (&PrometheusClusterValidator{}).SetupWithManager(mgr)
}
