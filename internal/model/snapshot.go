package model

import (
	"fmt"
	"time"
)

type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

type Snapshot struct {
	Version   string     `json:"version"`
	Cluster   string     `json:"cluster"`
	Namespace string     `json:"namespace"`
	CreatedAt time.Time  `json:"created_at"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Kind        string            `json:"kind"`
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Deployment  *DeploymentState  `json:"deployment,omitempty"`
	ConfigMap   *ConfigMapState   `json:"config_map,omitempty"`
	Secret      *SecretState      `json:"secret,omitempty"`
	Service     *ServiceState     `json:"service,omitempty"`
	Ingress     *IngressState     `json:"ingress,omitempty"`
	HPA         *HPAState         `json:"hpa,omitempty"`
}

func (r Resource) Key() string {
	return fmt.Sprintf("%s/%s/%s", r.Kind, r.Namespace, r.Name)
}

type DeploymentState struct {
	Replicas   *int32            `json:"replicas,omitempty"`
	Selector   map[string]string `json:"selector,omitempty"`
	PodLabels  map[string]string `json:"pod_labels,omitempty"`
	Containers []ContainerState  `json:"containers,omitempty"`
}

type ContainerState struct {
	Name      string        `json:"name"`
	Image     string        `json:"image,omitempty"`
	Env       []EnvVarState `json:"env,omitempty"`
	Resources ResourceState `json:"resources,omitempty"`
}

type ResourceState struct {
	CPURequest    string `json:"cpu_request,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	MemoryRequest string `json:"memory_request,omitempty"`
	MemoryLimit   string `json:"memory_limit,omitempty"`
}

type EnvVarState struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Hash      string `json:"hash,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

type ConfigMapState struct {
	Data       map[string]string `json:"data,omitempty"`
	BinaryKeys []string          `json:"binary_keys,omitempty"`
}

type SecretState struct {
	Type        string            `json:"type,omitempty"`
	Immutable   *bool             `json:"immutable,omitempty"`
	Keys        []string          `json:"keys,omitempty"`
	ValueHashes map[string]string `json:"value_hashes,omitempty"`
}

type ServiceState struct {
	Type     string             `json:"type,omitempty"`
	Selector map[string]string  `json:"selector,omitempty"`
	Ports    []ServicePortState `json:"ports,omitempty"`
}

type ServicePortState struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port,omitempty"`
}

type IngressState struct {
	ClassName string             `json:"class_name,omitempty"`
	Rules     []IngressRuleState `json:"rules,omitempty"`
	TLSHosts  []string           `json:"tls_hosts,omitempty"`
}

type IngressRuleState struct {
	Host  string             `json:"host,omitempty"`
	Paths []IngressPathState `json:"paths,omitempty"`
}

type IngressPathState struct {
	Path        string `json:"path,omitempty"`
	PathType    string `json:"path_type,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
	ServicePort string `json:"service_port,omitempty"`
}

type HPAState struct {
	MinReplicas *int32           `json:"min_replicas,omitempty"`
	MaxReplicas int32            `json:"max_replicas"`
	Metrics     []HPAMetricState `json:"metrics,omitempty"`
}

type HPAMetricState struct {
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Target string `json:"target,omitempty"`
}

type Change struct {
	Risk         Risk   `json:"risk"`
	ResourceKind string `json:"resource_kind"`
	Namespace    string `json:"namespace"`
	ResourceName string `json:"resource_name"`
	Path         string `json:"path"`
	Before       string `json:"before,omitempty"`
	After        string `json:"after,omitempty"`
	Summary      string `json:"summary"`
}

type CompareResult struct {
	BeforePath  string    `json:"before_path"`
	AfterPath   string    `json:"after_path"`
	Namespace   string    `json:"namespace"`
	GeneratedAt time.Time `json:"generated_at"`
	Summary     Summary   `json:"summary"`
	Changes     []Change  `json:"changes"`
}

type Summary struct {
	Total  int `json:"total"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}
