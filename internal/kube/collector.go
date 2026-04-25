package kube

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

type Collector struct {
	kubeconfigPath string
}

func NewCollector(kubeconfigPath string) *Collector {
	return &Collector{kubeconfigPath: kubeconfigPath}
}

func (c *Collector) CollectSnapshot(ctx context.Context, namespace string) (model.Snapshot, error) {
	restConfig, rawConfig, _, err := LoadRESTConfig(c.kubeconfigPath)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("create kubernetes client: %w", err)
	}

	scope := namespace
	if scope == "" {
		scope = metav1.NamespaceAll
	}

	snapshot := model.Snapshot{
		Version:   "v1",
		Cluster:   CurrentClusterName(rawConfig),
		Namespace: displayNamespace(namespace),
		CreatedAt: time.Now().UTC(),
	}

	deployments, err := clientset.AppsV1().Deployments(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list deployments: %w", err)
	}
	for _, item := range deployments.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeDeployment(item))
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list configmaps: %w", err)
	}
	for _, item := range configMaps.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeConfigMap(item))
	}

	secrets, err := clientset.CoreV1().Secrets(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list secrets: %w", err)
	}
	for _, item := range secrets.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeSecret(item))
	}

	services, err := clientset.CoreV1().Services(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list services: %w", err)
	}
	for _, item := range services.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeService(item))
	}

	ingresses, err := clientset.NetworkingV1().Ingresses(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list ingresses: %w", err)
	}
	for _, item := range ingresses.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeIngress(item))
	}

	hpas, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list horizontalpodautoscalers: %w", err)
	}
	for _, item := range hpas.Items {
		snapshot.Resources = append(snapshot.Resources, normalizeHPA(item))
	}

	sort.Slice(snapshot.Resources, func(i, j int) bool {
		return snapshot.Resources[i].Key() < snapshot.Resources[j].Key()
	})

	return snapshot, nil
}

func normalizeDeployment(item appsv1.Deployment) model.Resource {
	selector := map[string]string(nil)
	if item.Spec.Selector != nil {
		selector = cloneMap(item.Spec.Selector.MatchLabels)
	}

	return model.Resource{
		Kind:      "Deployment",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		Deployment: &model.DeploymentState{
			Replicas:   item.Spec.Replicas,
			Selector:   selector,
			PodLabels:  cloneMap(item.Spec.Template.Labels),
			Containers: normalizeContainers(item.Spec.Template.Spec.Containers),
		},
	}
}

func normalizeContainers(items []corev1.Container) []model.ContainerState {
	containers := make([]model.ContainerState, 0, len(items))
	for _, item := range items {
		containers = append(containers, model.ContainerState{
			Name:  item.Name,
			Image: item.Image,
			Env:   normalizeEnvVars(item.Env),
			Resources: model.ResourceState{
				CPURequest:    item.Resources.Requests.Cpu().String(),
				CPULimit:      item.Resources.Limits.Cpu().String(),
				MemoryRequest: item.Resources.Requests.Memory().String(),
				MemoryLimit:   item.Resources.Limits.Memory().String(),
			},
		})
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

func normalizeEnvVars(items []corev1.EnvVar) []model.EnvVarState {
	out := make([]model.EnvVarState, 0, len(items))
	for _, item := range items {
		env := model.EnvVarState{Name: item.Name}

		switch {
		case item.ValueFrom == nil:
			env.Type = "literal"
			if isSensitiveName(item.Name) {
				env.Hash = hashString(item.Value)
				env.Sensitive = true
			} else {
				env.Value = item.Value
			}
		case item.ValueFrom.SecretKeyRef != nil:
			env.Type = "secretKeyRef"
			env.Ref = fmt.Sprintf("%s/%s", item.ValueFrom.SecretKeyRef.Name, item.ValueFrom.SecretKeyRef.Key)
			env.Sensitive = true
		case item.ValueFrom.ConfigMapKeyRef != nil:
			env.Type = "configMapKeyRef"
			env.Ref = fmt.Sprintf("%s/%s", item.ValueFrom.ConfigMapKeyRef.Name, item.ValueFrom.ConfigMapKeyRef.Key)
		case item.ValueFrom.FieldRef != nil:
			env.Type = "fieldRef"
			env.Ref = item.ValueFrom.FieldRef.FieldPath
		case item.ValueFrom.ResourceFieldRef != nil:
			env.Type = "resourceFieldRef"
			env.Ref = item.ValueFrom.ResourceFieldRef.Resource
		default:
			env.Type = "derived"
		}

		out = append(out, env)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}

func normalizeConfigMap(item corev1.ConfigMap) model.Resource {
	data := make(map[string]string, len(item.Data))
	for key, value := range item.Data {
		data[key] = normalizeConfigValue(key, value)
	}

	binaryKeys := make([]string, 0, len(item.BinaryData))
	for key := range item.BinaryData {
		binaryKeys = append(binaryKeys, key)
	}
	sort.Strings(binaryKeys)

	return model.Resource{
		Kind:      "ConfigMap",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		ConfigMap: &model.ConfigMapState{
			Data:       data,
			BinaryKeys: binaryKeys,
		},
	}
}

func normalizeSecret(item corev1.Secret) model.Resource {
	keys := make([]string, 0, len(item.Data))
	hashes := make(map[string]string, len(item.Data))
	for key, value := range item.Data {
		keys = append(keys, key)
		hashes[key] = hashBytes(value)
	}
	sort.Strings(keys)

	return model.Resource{
		Kind:      "Secret",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		Secret: &model.SecretState{
			Type:        string(item.Type),
			Immutable:   item.Immutable,
			Keys:        keys,
			ValueHashes: hashes,
		},
	}
}

func normalizeService(item corev1.Service) model.Resource {
	ports := make([]model.ServicePortState, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		ports = append(ports, model.ServicePortState{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort.String(),
		})
	}

	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Name != ports[j].Name {
			return ports[i].Name < ports[j].Name
		}
		return ports[i].Port < ports[j].Port
	})

	return model.Resource{
		Kind:      "Service",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		Service: &model.ServiceState{
			Type:     string(item.Spec.Type),
			Selector: cloneMap(item.Spec.Selector),
			Ports:    ports,
		},
	}
}

func normalizeIngress(item networkingv1.Ingress) model.Resource {
	rules := make([]model.IngressRuleState, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		state := model.IngressRuleState{Host: rule.Host}
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				pathType := ""
				if path.PathType != nil {
					pathType = string(*path.PathType)
				}

				serviceName := ""
				servicePort := ""
				if path.Backend.Service != nil {
					serviceName = path.Backend.Service.Name
					servicePort = ingressServicePort(path.Backend.Service.Port)
				} else if path.Backend.Resource != nil {
					serviceName = path.Backend.Resource.Name
				}

				state.Paths = append(state.Paths, model.IngressPathState{
					Path:        path.Path,
					PathType:    pathType,
					ServiceName: serviceName,
					ServicePort: servicePort,
				})
			}
			sort.Slice(state.Paths, func(i, j int) bool {
				return state.Paths[i].Path < state.Paths[j].Path
			})
		}
		rules = append(rules, state)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Host < rules[j].Host
	})

	tlsHosts := make([]string, 0)
	tlsIndex := make(map[string]struct{})
	for _, tls := range item.Spec.TLS {
		for _, host := range tls.Hosts {
			if _, found := tlsIndex[host]; found {
				continue
			}
			tlsIndex[host] = struct{}{}
			tlsHosts = append(tlsHosts, host)
		}
	}
	sort.Strings(tlsHosts)

	className := ""
	if item.Spec.IngressClassName != nil {
		className = *item.Spec.IngressClassName
	}

	return model.Resource{
		Kind:      "Ingress",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		Ingress: &model.IngressState{
			ClassName: className,
			Rules:     rules,
			TLSHosts:  tlsHosts,
		},
	}
}

func normalizeHPA(item autoscalingv2.HorizontalPodAutoscaler) model.Resource {
	metrics := make([]model.HPAMetricState, 0, len(item.Spec.Metrics))
	for _, metric := range item.Spec.Metrics {
		metrics = append(metrics, normalizeMetric(metric))
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Type != metrics[j].Type {
			return metrics[i].Type < metrics[j].Type
		}
		return metrics[i].Name < metrics[j].Name
	})

	return model.Resource{
		Kind:      "HorizontalPodAutoscaler",
		Namespace: item.Namespace,
		Name:      item.Name,
		Labels:    cloneMap(item.Labels),
		HPA: &model.HPAState{
			MinReplicas: item.Spec.MinReplicas,
			MaxReplicas: item.Spec.MaxReplicas,
			Metrics:     metrics,
		},
	}
}

func normalizeMetric(metric autoscalingv2.MetricSpec) model.HPAMetricState {
	state := model.HPAMetricState{Type: string(metric.Type)}

	switch metric.Type {
	case autoscalingv2.ResourceMetricSourceType:
		if metric.Resource == nil {
			return state
		}
		state.Name = string(metric.Resource.Name)
		state.Target = formatMetricTarget(metric.Resource.Target)
	case autoscalingv2.ContainerResourceMetricSourceType:
		if metric.ContainerResource == nil {
			return state
		}
		state.Name = string(metric.ContainerResource.Name) + "/" + metric.ContainerResource.Container
		state.Target = formatMetricTarget(metric.ContainerResource.Target)
	case autoscalingv2.PodsMetricSourceType:
		if metric.Pods == nil {
			return state
		}
		state.Name = metric.Pods.Metric.Name
		state.Target = formatMetricTarget(metric.Pods.Target)
	case autoscalingv2.ObjectMetricSourceType:
		if metric.Object == nil {
			return state
		}
		state.Name = metric.Object.Metric.Name + "@" + metric.Object.DescribedObject.Kind + "/" + metric.Object.DescribedObject.Name
		state.Target = formatMetricTarget(metric.Object.Target)
	case autoscalingv2.ExternalMetricSourceType:
		if metric.External == nil {
			return state
		}
		state.Name = metric.External.Metric.Name
		state.Target = formatMetricTarget(metric.External.Target)
	}

	return state
}

func formatMetricTarget(target autoscalingv2.MetricTarget) string {
	parts := []string{string(target.Type)}
	if target.Value != nil {
		parts = append(parts, target.Value.String())
	}
	if target.AverageValue != nil {
		parts = append(parts, "avg="+target.AverageValue.String())
	}
	if target.AverageUtilization != nil {
		parts = append(parts, fmt.Sprintf("util=%d%%", *target.AverageUtilization))
	}
	return strings.Join(parts, ":")
}

func normalizeConfigValue(key, value string) string {
	if isSensitiveName(key) {
		return "redacted(" + hashString(value) + ")"
	}
	return value
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func isSensitiveName(value string) bool {
	normalized := strings.ToLower(value)
	fragments := []string{"password", "secret", "token", "apikey", "api_key", "private_key", "client_secret", "access_key"}
	for _, fragment := range fragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func cloneMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func displayNamespace(namespace string) string {
	if namespace == "" {
		return "all"
	}
	return namespace
}

func ingressServicePort(port networkingv1.ServiceBackendPort) string {
	if port.Name != "" {
		return port.Name
	}
	return fmt.Sprintf("%d", port.Number)
}
