package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/asobitov2005/OpsDiff/internal/model"
	timelineutil "github.com/asobitov2005/OpsDiff/internal/timeline"
)

func (c *Collector) CollectTimeline(ctx context.Context, namespace string, from time.Duration, limit int) (model.Timeline, error) {
	restConfig, rawConfig, _, err := LoadRESTConfig(c.kubeconfigPath)
	if err != nil {
		return model.Timeline{}, fmt.Errorf("load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return model.Timeline{}, fmt.Errorf("create kubernetes client: %w", err)
	}

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-from)

	scope := namespace
	if scope == "" {
		scope = metav1.NamespaceAll
	}

	events, err := clientset.CoreV1().Events(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Timeline{}, fmt.Errorf("list kubernetes events: %w", err)
	}

	pods, err := clientset.CoreV1().Pods(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return model.Timeline{}, fmt.Errorf("list pods: %w", err)
	}

	items := normalizeClusterEvents(events.Items, windowStart)
	items = append(items, podSignalEvents(pods.Items, windowStart, windowEnd)...)

	return timelineutil.Build(CurrentClusterName(rawConfig), displayNamespace(namespace), windowStart, windowEnd, items, limit), nil
}

func normalizeClusterEvents(items []corev1.Event, windowStart time.Time) []model.TimelineEvent {
	out := make([]model.TimelineEvent, 0, len(items))
	for _, item := range items {
		event, ok := normalizeClusterEvent(item, windowStart)
		if ok {
			out = append(out, event)
		}
	}
	return out
}

func normalizeClusterEvent(item corev1.Event, windowStart time.Time) (model.TimelineEvent, bool) {
	if !includeClusterEvent(item) {
		return model.TimelineEvent{}, false
	}

	timestamp := eventTimestamp(item)
	if timestamp.IsZero() || timestamp.Before(windowStart) {
		return model.TimelineEvent{}, false
	}

	category := "change"
	severity := "info"
	riskTags := riskTagsForReason(item.Reason)

	if item.Type == corev1.EventTypeWarning {
		category = "symptom"
		severity = "warning"
	}

	if hasRiskTag(riskTags, "crashloop") || hasRiskTag(riskTags, "oomkilled") {
		severity = "critical"
	}

	message := compactMessage(item.Message)
	if item.Count > 1 {
		message = fmt.Sprintf("%s (count=%d)", message, item.Count)
	}

	return model.TimelineEvent{
		Time:         timestamp.UTC(),
		Source:       "kubernetes",
		Category:     category,
		Severity:     severity,
		Namespace:    item.Namespace,
		Service:      inferServiceFromObject(item.InvolvedObject.Kind, item.InvolvedObject.Name),
		ResourceKind: item.InvolvedObject.Kind,
		ResourceName: item.InvolvedObject.Name,
		Reason:       item.Reason,
		Message:      message,
		RiskTags:     riskTags,
	}, true
}

func podSignalEvents(items []corev1.Pod, windowStart, observedAt time.Time) []model.TimelineEvent {
	out := make([]model.TimelineEvent, 0)
	for _, pod := range items {
		out = append(out, podContainerSignalEvents(pod, windowStart, observedAt)...)
	}
	return out
}

func podContainerSignalEvents(pod corev1.Pod, windowStart, observedAt time.Time) []model.TimelineEvent {
	out := make([]model.TimelineEvent, 0)
	service := inferServiceName(pod.Labels, pod.Name, pod.OwnerReferences)

	for _, status := range pod.Status.ContainerStatuses {
		restartTime, restartKnown := latestRestartTime(status, pod, windowStart)
		if status.RestartCount > 0 && restartKnown && !restartTime.Before(windowStart) {
			severity := "warning"
			if status.RestartCount >= 3 {
				severity = "critical"
			}

			out = append(out, model.TimelineEvent{
				Time:         restartTime.UTC(),
				Source:       "kubernetes",
				Category:     "evidence",
				Severity:     severity,
				Namespace:    pod.Namespace,
				Service:      service,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Reason:       "ContainerRestarted",
				Message:      fmt.Sprintf("container %s restarted %d times", status.Name, status.RestartCount),
				RiskTags:     []string{"restart-evidence"},
			})
		}

		if termination := latestTerminatedState(status); termination != nil && termination.Reason == "OOMKilled" {
			timestamp := termination.FinishedAt.Time
			if timestamp.IsZero() {
				timestamp = fallbackPodTime(pod, observedAt)
			}

			if !timestamp.Before(windowStart) {
				out = append(out, model.TimelineEvent{
					Time:         timestamp.UTC(),
					Source:       "kubernetes",
					Category:     "symptom",
					Severity:     "critical",
					Namespace:    pod.Namespace,
					Service:      service,
					ResourceKind: "Pod",
					ResourceName: pod.Name,
					Reason:       "OOMKilled",
					Message:      fmt.Sprintf("container %s was OOMKilled", status.Name),
					RiskTags:     []string{"oomkilled", "restart-evidence"},
				})
			}
		}

		if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
			timestamp := observedAt
			if !restartTime.IsZero() && restartTime.After(windowStart) {
				timestamp = restartTime
			}

			out = append(out, model.TimelineEvent{
				Time:         timestamp.UTC(),
				Source:       "kubernetes",
				Category:     "symptom",
				Severity:     "critical",
				Namespace:    pod.Namespace,
				Service:      service,
				ResourceKind: "Pod",
				ResourceName: pod.Name,
				Reason:       "CrashLoopBackOff",
				Message:      fmt.Sprintf("container %s is waiting in CrashLoopBackOff", status.Name),
				RiskTags:     []string{"crashloop", "restart-evidence"},
			})
		}
	}

	return out
}

func includeClusterEvent(item corev1.Event) bool {
	if item.Type == corev1.EventTypeWarning {
		return true
	}

	normalReasons := map[string]struct{}{
		"ScalingReplicaSet": {},
		"SuccessfulCreate":  {},
		"SuccessfulDelete":  {},
		"Killing":           {},
		"Scheduled":         {},
	}

	_, ok := normalReasons[item.Reason]
	return ok
}

func eventTimestamp(item corev1.Event) time.Time {
	switch {
	case !item.EventTime.Time.IsZero():
		return item.EventTime.Time
	case item.Series != nil && !item.Series.LastObservedTime.Time.IsZero():
		return item.Series.LastObservedTime.Time
	case !item.LastTimestamp.Time.IsZero():
		return item.LastTimestamp.Time
	case !item.FirstTimestamp.Time.IsZero():
		return item.FirstTimestamp.Time
	default:
		return item.CreationTimestamp.Time
	}
}

func latestRestartTime(status corev1.ContainerStatus, pod corev1.Pod, windowStart time.Time) (time.Time, bool) {
	if terminated := latestTerminatedState(status); terminated != nil && !terminated.FinishedAt.Time.IsZero() {
		return terminated.FinishedAt.Time, true
	}

	if !pod.CreationTimestamp.Time.IsZero() && !pod.CreationTimestamp.Time.Before(windowStart) {
		return pod.CreationTimestamp.Time, true
	}

	return time.Time{}, false
}

func latestTerminatedState(status corev1.ContainerStatus) *corev1.ContainerStateTerminated {
	switch {
	case status.State.Terminated != nil:
		return status.State.Terminated
	case status.LastTerminationState.Terminated != nil:
		return status.LastTerminationState.Terminated
	default:
		return nil
	}
}

func fallbackPodTime(pod corev1.Pod, observedAt time.Time) time.Time {
	if !pod.CreationTimestamp.Time.IsZero() {
		return pod.CreationTimestamp.Time
	}
	return observedAt
}

func riskTagsForReason(reason string) []string {
	switch reason {
	case "ScalingReplicaSet", "SuccessfulCreate", "SuccessfulDelete", "Killing", "Scheduled":
		return []string{"rollout-evidence"}
	case "BackOff":
		return []string{"crashloop", "restart-evidence"}
	case "Unhealthy":
		return []string{"probe-failure"}
	case "FailedMount":
		return []string{"storage-impact"}
	case "FailedScheduling":
		return []string{"capacity-impact"}
	default:
		return nil
	}
}

func hasRiskTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func compactMessage(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func inferServiceName(labels map[string]string, objectName string, owners []metav1.OwnerReference) string {
	labelKeys := []string{
		"app.kubernetes.io/name",
		"app.kubernetes.io/instance",
		"app",
		"k8s-app",
		"service",
	}
	for _, key := range labelKeys {
		if labels[key] != "" {
			return labels[key]
		}
	}

	for _, owner := range owners {
		if owner.Name != "" {
			return inferServiceFromObject(owner.Kind, owner.Name)
		}
	}

	return inferServiceFromObject("Pod", objectName)
}

func inferServiceFromObject(kind, name string) string {
	if name == "" {
		return ""
	}

	switch kind {
	case "Pod":
		return trimGeneratedName(name, 2)
	case "ReplicaSet", "Job":
		return trimGeneratedName(name, 1)
	case "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress", "ConfigMap", "Secret":
		return name
	default:
		return name
	}
}

func trimGeneratedName(name string, suffixParts int) string {
	parts := strings.Split(name, "-")
	if len(parts) <= suffixParts {
		return name
	}
	return strings.Join(parts[:len(parts)-suffixParts], "-")
}
