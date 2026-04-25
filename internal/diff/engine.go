package diff

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Engine struct{}

func NewEngine() Engine {
	return Engine{}
}

func (Engine) Compare(before, after model.Snapshot, beforePath, afterPath string) model.CompareResult {
	beforeResources := make(map[string]model.Resource, len(before.Resources))
	afterResources := make(map[string]model.Resource, len(after.Resources))
	union := make(map[string]struct{}, len(before.Resources)+len(after.Resources))

	for _, resource := range before.Resources {
		beforeResources[resource.Key()] = resource
		union[resource.Key()] = struct{}{}
	}

	for _, resource := range after.Resources {
		afterResources[resource.Key()] = resource
		union[resource.Key()] = struct{}{}
	}

	keys := make([]string, 0, len(union))
	for key := range union {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	changes := make([]model.Change, 0)
	for _, key := range keys {
		beforeResource, beforeFound := beforeResources[key]
		afterResource, afterFound := afterResources[key]

		switch {
		case !beforeFound && afterFound:
			changes = append(changes, model.Change{
				Risk:         lifecycleRisk(afterResource.Kind),
				ResourceKind: afterResource.Kind,
				Namespace:    afterResource.Namespace,
				ResourceName: afterResource.Name,
				Path:         "resource",
				After:        "created",
				Summary:      fmt.Sprintf("%s/%s created", afterResource.Kind, afterResource.Name),
			})
		case beforeFound && !afterFound:
			changes = append(changes, model.Change{
				Risk:         lifecycleRisk(beforeResource.Kind),
				ResourceKind: beforeResource.Kind,
				Namespace:    beforeResource.Namespace,
				ResourceName: beforeResource.Name,
				Path:         "resource",
				Before:       "present",
				After:        "deleted",
				Summary:      fmt.Sprintf("%s/%s deleted", beforeResource.Kind, beforeResource.Name),
			})
		default:
			changes = append(changes, diffResource(beforeResource, afterResource)...)
		}
	}

	sort.SliceStable(changes, func(i, j int) bool {
		left := changes[i]
		right := changes[j]
		if riskWeight(left.Risk) != riskWeight(right.Risk) {
			return riskWeight(left.Risk) > riskWeight(right.Risk)
		}
		if left.ResourceKind != right.ResourceKind {
			return left.ResourceKind < right.ResourceKind
		}
		if left.ResourceName != right.ResourceName {
			return left.ResourceName < right.ResourceName
		}
		return left.Path < right.Path
	})

	result := model.CompareResult{
		BeforePath:  beforePath,
		AfterPath:   afterPath,
		Namespace:   after.Namespace,
		GeneratedAt: time.Now().UTC(),
		Changes:     changes,
	}
	result.Summary = summarize(changes)
	return result
}

func diffResource(before, after model.Resource) []model.Change {
	switch {
	case before.Deployment != nil && after.Deployment != nil:
		return diffDeployment(before, after)
	case before.ConfigMap != nil && after.ConfigMap != nil:
		return diffConfigMap(before, after)
	case before.Secret != nil && after.Secret != nil:
		return diffSecret(before, after)
	case before.Service != nil && after.Service != nil:
		return diffService(before, after)
	case before.Ingress != nil && after.Ingress != nil:
		return diffIngress(before, after)
	case before.HPA != nil && after.HPA != nil:
		return diffHPA(before, after)
	default:
		return nil
	}
}

func diffDeployment(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	previous := before.Deployment
	current := after.Deployment

	if stringifyInt32Ptr(previous.Replicas) != stringifyInt32Ptr(current.Replicas) {
		changes = append(changes, change(after, "spec.replicas", model.RiskMedium, stringifyInt32Ptr(previous.Replicas), stringifyInt32Ptr(current.Replicas), "replica count changed"))
	}

	beforeContainers := indexContainers(previous.Containers)
	afterContainers := indexContainers(current.Containers)
	for _, name := range unionKeys(beforeContainers, afterContainers) {
		left, leftOK := beforeContainers[name]
		right, rightOK := afterContainers[name]
		basePath := "spec.template.spec.containers." + name

		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, basePath, model.RiskHigh, "", "added", fmt.Sprintf("container %s added", name)))
			continue
		case leftOK && !rightOK:
			changes = append(changes, change(after, basePath, model.RiskHigh, "present", "removed", fmt.Sprintf("container %s removed", name)))
			continue
		}

		if left.Image != right.Image {
			changes = append(changes, change(after, basePath+".image", model.RiskMedium, left.Image, right.Image, fmt.Sprintf("container %s image changed", name)))
		}

		changes = append(changes, diffResources(after, basePath+".resources", left.Resources, right.Resources)...)
		changes = append(changes, diffEnv(after, basePath+".env", left.Env, right.Env)...)
	}

	return changes
}

func diffResources(resource model.Resource, basePath string, before, after model.ResourceState) []model.Change {
	changes := make([]model.Change, 0)
	if before.CPURequest != after.CPURequest {
		changes = append(changes, change(resource, basePath+".requests.cpu", resourceRiskForCapacity(before.CPURequest, after.CPURequest), before.CPURequest, after.CPURequest, "CPU request changed"))
	}
	if before.CPULimit != after.CPULimit {
		changes = append(changes, change(resource, basePath+".limits.cpu", resourceRiskForCapacity(before.CPULimit, after.CPULimit), before.CPULimit, after.CPULimit, "CPU limit changed"))
	}
	if before.MemoryRequest != after.MemoryRequest {
		changes = append(changes, change(resource, basePath+".requests.memory", resourceRiskForCapacity(before.MemoryRequest, after.MemoryRequest), before.MemoryRequest, after.MemoryRequest, "memory request changed"))
	}
	if before.MemoryLimit != after.MemoryLimit {
		changes = append(changes, change(resource, basePath+".limits.memory", resourceRiskForCapacity(before.MemoryLimit, after.MemoryLimit), before.MemoryLimit, after.MemoryLimit, "memory limit changed"))
	}
	return changes
}

func diffEnv(resource model.Resource, basePath string, before, after []model.EnvVarState) []model.Change {
	changes := make([]model.Change, 0)
	beforeEnv := indexEnv(before)
	afterEnv := indexEnv(after)

	for _, name := range unionKeys(beforeEnv, afterEnv) {
		left, leftOK := beforeEnv[name]
		right, rightOK := afterEnv[name]
		path := basePath + "." + name
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(resource, path, envRisk(name), "", envDisplay(right), fmt.Sprintf("env %s added", name)))
		case leftOK && !rightOK:
			changes = append(changes, change(resource, path, envRisk(name), envDisplay(left), "", fmt.Sprintf("env %s removed", name)))
		default:
			if envDisplay(left) != envDisplay(right) {
				changes = append(changes, change(resource, path, envRisk(name), envDisplay(left), envDisplay(right), fmt.Sprintf("env %s changed", name)))
			}
		}
	}

	return changes
}

func diffConfigMap(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	for _, key := range unionStringMapKeys(before.ConfigMap.Data, after.ConfigMap.Data) {
		left, leftOK := before.ConfigMap.Data[key]
		right, rightOK := after.ConfigMap.Data[key]

		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, "data."+key, configRisk(key), "", right, fmt.Sprintf("config key %s added", key)))
		case leftOK && !rightOK:
			changes = append(changes, change(after, "data."+key, configRisk(key), left, "", fmt.Sprintf("config key %s removed", key)))
		case left != right:
			changes = append(changes, change(after, "data."+key, configRisk(key), left, right, fmt.Sprintf("config key %s changed", key)))
		}
	}

	for _, key := range unionStringSlices(before.ConfigMap.BinaryKeys, after.ConfigMap.BinaryKeys) {
		beforeHas := contains(before.ConfigMap.BinaryKeys, key)
		afterHas := contains(after.ConfigMap.BinaryKeys, key)
		switch {
		case !beforeHas && afterHas:
			changes = append(changes, change(after, "binaryData."+key, model.RiskMedium, "", "present", fmt.Sprintf("binary config key %s added", key)))
		case beforeHas && !afterHas:
			changes = append(changes, change(after, "binaryData."+key, model.RiskMedium, "present", "", fmt.Sprintf("binary config key %s removed", key)))
		}
	}

	return changes
}

func diffSecret(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	for _, key := range unionStringMapKeys(before.Secret.ValueHashes, after.Secret.ValueHashes) {
		left, leftOK := before.Secret.ValueHashes[key]
		right, rightOK := after.Secret.ValueHashes[key]
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, "data."+key, model.RiskHigh, "", right, fmt.Sprintf("secret key %s added", key)))
		case leftOK && !rightOK:
			changes = append(changes, change(after, "data."+key, model.RiskHigh, left, "", fmt.Sprintf("secret key %s removed", key)))
		case left != right:
			changes = append(changes, change(after, "data."+key, model.RiskHigh, left, right, fmt.Sprintf("secret key %s changed", key)))
		}
	}
	return changes
}

func diffService(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	if before.Service.Type != after.Service.Type {
		changes = append(changes, change(after, "spec.type", model.RiskMedium, before.Service.Type, after.Service.Type, "service type changed"))
	}

	for _, key := range unionStringMapKeys(before.Service.Selector, after.Service.Selector) {
		left, leftOK := before.Service.Selector[key]
		right, rightOK := after.Service.Selector[key]
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, "spec.selector."+key, model.RiskHigh, "", right, fmt.Sprintf("service selector %s added", key)))
		case leftOK && !rightOK:
			changes = append(changes, change(after, "spec.selector."+key, model.RiskHigh, left, "", fmt.Sprintf("service selector %s removed", key)))
		case left != right:
			changes = append(changes, change(after, "spec.selector."+key, model.RiskHigh, left, right, fmt.Sprintf("service selector %s changed", key)))
		}
	}

	beforePorts := indexPorts(before.Service.Ports)
	afterPorts := indexPorts(after.Service.Ports)
	for _, key := range unionKeys(beforePorts, afterPorts) {
		left, leftOK := beforePorts[key]
		right, rightOK := afterPorts[key]
		path := "spec.ports." + key
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, path, model.RiskMedium, "", portDisplay(right), fmt.Sprintf("service port %s added", key)))
		case leftOK && !rightOK:
			changes = append(changes, change(after, path, model.RiskMedium, portDisplay(left), "", fmt.Sprintf("service port %s removed", key)))
		case portDisplay(left) != portDisplay(right):
			changes = append(changes, change(after, path, model.RiskMedium, portDisplay(left), portDisplay(right), fmt.Sprintf("service port %s changed", key)))
		}
	}

	return changes
}

func diffIngress(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	if before.Ingress.ClassName != after.Ingress.ClassName {
		changes = append(changes, change(after, "spec.ingressClassName", model.RiskMedium, before.Ingress.ClassName, after.Ingress.ClassName, "ingress class changed"))
	}

	beforeRules := flattenIngressRules(before.Ingress.Rules)
	afterRules := flattenIngressRules(after.Ingress.Rules)
	for _, key := range unionKeys(beforeRules, afterRules) {
		left, leftOK := beforeRules[key]
		right, rightOK := afterRules[key]
		path := "spec.rules." + key
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, path, model.RiskHigh, "", right, "ingress route added"))
		case leftOK && !rightOK:
			changes = append(changes, change(after, path, model.RiskHigh, left, "", "ingress route removed"))
		case left != right:
			changes = append(changes, change(after, path, model.RiskHigh, left, right, "ingress route changed"))
		}
	}

	for _, host := range unionStringSlices(before.Ingress.TLSHosts, after.Ingress.TLSHosts) {
		beforeHas := contains(before.Ingress.TLSHosts, host)
		afterHas := contains(after.Ingress.TLSHosts, host)
		switch {
		case !beforeHas && afterHas:
			changes = append(changes, change(after, "spec.tls."+host, model.RiskLow, "", "enabled", fmt.Sprintf("TLS host %s added", host)))
		case beforeHas && !afterHas:
			changes = append(changes, change(after, "spec.tls."+host, model.RiskLow, "enabled", "", fmt.Sprintf("TLS host %s removed", host)))
		}
	}

	return changes
}

func diffHPA(before, after model.Resource) []model.Change {
	changes := make([]model.Change, 0)
	if stringifyInt32Ptr(before.HPA.MinReplicas) != stringifyInt32Ptr(after.HPA.MinReplicas) {
		changes = append(changes, change(after, "spec.minReplicas", model.RiskMedium, stringifyInt32Ptr(before.HPA.MinReplicas), stringifyInt32Ptr(after.HPA.MinReplicas), "HPA min replicas changed"))
	}
	if before.HPA.MaxReplicas != after.HPA.MaxReplicas {
		changes = append(changes, change(after, "spec.maxReplicas", model.RiskHigh, fmt.Sprintf("%d", before.HPA.MaxReplicas), fmt.Sprintf("%d", after.HPA.MaxReplicas), "HPA max replicas changed"))
	}

	beforeMetrics := indexMetrics(before.HPA.Metrics)
	afterMetrics := indexMetrics(after.HPA.Metrics)
	for _, key := range unionKeys(beforeMetrics, afterMetrics) {
		left, leftOK := beforeMetrics[key]
		right, rightOK := afterMetrics[key]
		path := "spec.metrics." + key
		switch {
		case !leftOK && rightOK:
			changes = append(changes, change(after, path, model.RiskMedium, "", metricDisplay(right), fmt.Sprintf("HPA metric %s added", key)))
		case leftOK && !rightOK:
			changes = append(changes, change(after, path, model.RiskMedium, metricDisplay(left), "", fmt.Sprintf("HPA metric %s removed", key)))
		case metricDisplay(left) != metricDisplay(right):
			changes = append(changes, change(after, path, model.RiskMedium, metricDisplay(left), metricDisplay(right), fmt.Sprintf("HPA metric %s changed", key)))
		}
	}

	return changes
}

func change(resource model.Resource, path string, risk model.Risk, before, after, summary string) model.Change {
	return model.Change{
		Risk:         risk,
		ResourceKind: resource.Kind,
		Namespace:    resource.Namespace,
		ResourceName: resource.Name,
		Path:         path,
		Before:       before,
		After:        after,
		Summary:      summary,
	}
}

func summarize(changes []model.Change) model.Summary {
	summary := model.Summary{Total: len(changes)}
	for _, item := range changes {
		switch item.Risk {
		case model.RiskHigh:
			summary.High++
		case model.RiskMedium:
			summary.Medium++
		default:
			summary.Low++
		}
	}
	return summary
}

func lifecycleRisk(kind string) model.Risk {
	switch kind {
	case "Deployment", "Service", "Ingress", "HorizontalPodAutoscaler":
		return model.RiskHigh
	default:
		return model.RiskMedium
	}
}

func envRisk(name string) model.Risk {
	if isHotPathKey(name) || isSensitiveKey(name) {
		return model.RiskHigh
	}
	return model.RiskMedium
}

func configRisk(name string) model.Risk {
	if isHotPathKey(name) || isSensitiveKey(name) {
		return model.RiskHigh
	}
	return model.RiskMedium
}

func resourceRiskForCapacity(before, after string) model.Risk {
	if before != "" && after == "" {
		return model.RiskMedium
	}
	if before == "" && after != "" {
		return model.RiskMedium
	}
	if appearsReduced(before, after) {
		return model.RiskHigh
	}
	return model.RiskMedium
}

func appearsReduced(before, after string) bool {
	if before == "" || after == "" {
		return false
	}

	beforeQuantity, beforeErr := resource.ParseQuantity(before)
	afterQuantity, afterErr := resource.ParseQuantity(after)
	if beforeErr == nil && afterErr == nil {
		return afterQuantity.Cmp(beforeQuantity) < 0
	}

	return len(after) < len(before)
}

func isSensitiveKey(value string) bool {
	normalized := strings.ToLower(value)
	sensitiveFragments := []string{"password", "secret", "token", "apikey", "api_key", "private_key", "client_secret", "access_key"}
	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func isHotPathKey(value string) bool {
	normalized := strings.ToLower(value)
	hotPathFragments := []string{"db_", "database", "pool", "timeout", "limit", "cache", "connection", "ingress", "host", "port", "memory", "cpu"}
	for _, fragment := range hotPathFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func riskWeight(risk model.Risk) int {
	switch risk {
	case model.RiskHigh:
		return 3
	case model.RiskMedium:
		return 2
	default:
		return 1
	}
}

func stringifyInt32Ptr(value *int32) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func indexContainers(items []model.ContainerState) map[string]model.ContainerState {
	index := make(map[string]model.ContainerState, len(items))
	for _, item := range items {
		index[item.Name] = item
	}
	return index
}

func indexEnv(items []model.EnvVarState) map[string]model.EnvVarState {
	index := make(map[string]model.EnvVarState, len(items))
	for _, item := range items {
		index[item.Name] = item
	}
	return index
}

func indexPorts(items []model.ServicePortState) map[string]model.ServicePortState {
	index := make(map[string]model.ServicePortState, len(items))
	for _, item := range items {
		key := item.Name
		if key == "" {
			key = fmt.Sprintf("%s/%d", item.Protocol, item.Port)
		}
		index[key] = item
	}
	return index
}

func indexMetrics(items []model.HPAMetricState) map[string]model.HPAMetricState {
	index := make(map[string]model.HPAMetricState, len(items))
	for _, item := range items {
		index[item.Type+"/"+item.Name] = item
	}
	return index
}

func envDisplay(item model.EnvVarState) string {
	switch {
	case item.Hash != "":
		return "redacted(" + item.Hash + ")"
	case item.Ref != "":
		return item.Type + ":" + item.Ref
	default:
		return item.Value
	}
}

func portDisplay(item model.ServicePortState) string {
	return fmt.Sprintf("%s %d -> %s", item.Protocol, item.Port, item.TargetPort)
}

func metricDisplay(item model.HPAMetricState) string {
	if item.Target == "" {
		return item.Type + ":" + item.Name
	}
	return item.Type + ":" + item.Name + "=" + item.Target
}

func flattenIngressRules(items []model.IngressRuleState) map[string]string {
	flattened := make(map[string]string)
	for _, rule := range items {
		for _, path := range rule.Paths {
			key := rule.Host + path.Path
			flattened[key] = fmt.Sprintf("%s:%s", path.ServiceName, path.ServicePort)
		}
	}
	return flattened
}

func unionKeys[T any](left, right map[string]T) []string {
	union := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		union[key] = struct{}{}
	}
	for key := range right {
		union[key] = struct{}{}
	}
	keys := make([]string, 0, len(union))
	for key := range union {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func unionStringMapKeys(left, right map[string]string) []string {
	return unionKeys(left, right)
}

func unionStringSlices(left, right []string) []string {
	union := make(map[string]struct{}, len(left)+len(right))
	for _, item := range left {
		union[item] = struct{}{}
	}
	for _, item := range right {
		union[item] = struct{}{}
	}
	out := make([]string, 0, len(union))
	for item := range union {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
