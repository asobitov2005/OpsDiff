package diff

import (
	"testing"
	"time"

	"github.com/opsdiff/opsdiff/internal/model"
)

func TestCompareDetectsHighRiskDeploymentAndConfigChanges(t *testing.T) {
	before := model.Snapshot{
		Version:   "v1",
		Cluster:   "prod",
		Namespace: "prod",
		CreatedAt: time.Unix(0, 0).UTC(),
		Resources: []model.Resource{
			{
				Kind:      "Deployment",
				Namespace: "prod",
				Name:      "api",
				Deployment: &model.DeploymentState{
					Replicas: int32Ptr(3),
					Containers: []model.ContainerState{
						{
							Name:  "api",
							Image: "api:v1.8.2",
							Env: []model.EnvVarState{
								{Name: "DB_POOL_SIZE", Type: "literal", Value: "20"},
							},
							Resources: model.ResourceState{
								MemoryLimit: "512Mi",
							},
						},
					},
				},
			},
			{
				Kind:      "ConfigMap",
				Namespace: "prod",
				Name:      "api-config",
				ConfigMap: &model.ConfigMapState{
					Data: map[string]string{
						"DB_POOL_SIZE": "20",
					},
				},
			},
		},
	}

	after := model.Snapshot{
		Version:   "v1",
		Cluster:   "prod",
		Namespace: "prod",
		CreatedAt: time.Unix(60, 0).UTC(),
		Resources: []model.Resource{
			{
				Kind:      "Deployment",
				Namespace: "prod",
				Name:      "api",
				Deployment: &model.DeploymentState{
					Replicas: int32Ptr(3),
					Containers: []model.ContainerState{
						{
							Name:  "api",
							Image: "api:v1.8.3",
							Env: []model.EnvVarState{
								{Name: "DB_POOL_SIZE", Type: "literal", Value: "100"},
							},
							Resources: model.ResourceState{
								MemoryLimit: "256Mi",
							},
						},
					},
				},
			},
			{
				Kind:      "ConfigMap",
				Namespace: "prod",
				Name:      "api-config",
				ConfigMap: &model.ConfigMapState{
					Data: map[string]string{
						"DB_POOL_SIZE": "100",
					},
				},
			},
		},
	}

	result := NewEngine().Compare(before, after, "before.json", "after.json")
	if result.Summary.Total != 4 {
		t.Fatalf("expected 4 changes, got %d", result.Summary.Total)
	}

	expectRisk(t, result.Changes, "Deployment", "api", "spec.template.spec.containers.api.resources.limits.memory", model.RiskHigh)
	expectRisk(t, result.Changes, "Deployment", "api", "spec.template.spec.containers.api.env.DB_POOL_SIZE", model.RiskHigh)
	expectRisk(t, result.Changes, "ConfigMap", "api-config", "data.DB_POOL_SIZE", model.RiskHigh)
	expectRisk(t, result.Changes, "Deployment", "api", "spec.template.spec.containers.api.image", model.RiskMedium)
}

func expectRisk(t *testing.T, changes []model.Change, kind, name, path string, risk model.Risk) {
	t.Helper()
	for _, change := range changes {
		if change.ResourceKind == kind && change.ResourceName == name && change.Path == path {
			if change.Risk != risk {
				t.Fatalf("expected %s risk for %s/%s %s, got %s", risk, kind, name, path, change.Risk)
			}
			return
		}
	}

	t.Fatalf("missing change %s/%s %s", kind, name, path)
}

func int32Ptr(value int32) *int32 {
	return &value
}
