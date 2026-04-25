package prometheus

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

type alertEnvelope struct {
	Alerts []alertPayload `json:"alerts"`
}

type alertPayload struct {
	Status      string            `json:"status"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func LoadTimelineEvents(path string, windowStart, windowEnd time.Time) ([]model.TimelineEvent, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prometheus alerts %q: %w", path, err)
	}

	var alerts []alertPayload
	if err := json.Unmarshal(data, &alerts); err != nil {
		var envelope alertEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, fmt.Errorf("decode prometheus alerts %q: %w", path, err)
		}
		alerts = envelope.Alerts
	}

	events := make([]model.TimelineEvent, 0, len(alerts))
	for _, alert := range alerts {
		event, ok, err := normalizeAlert(alert, windowStart, windowEnd)
		if err != nil {
			return nil, err
		}
		if ok {
			events = append(events, event)
		}
	}

	return events, nil
}

func normalizeAlert(alert alertPayload, windowStart, windowEnd time.Time) (model.TimelineEvent, bool, error) {
	timestamp, err := parseAlertTime(alert.StartsAt)
	if err != nil {
		return model.TimelineEvent{}, false, err
	}
	if timestamp.Before(windowStart) || timestamp.After(windowEnd) {
		return model.TimelineEvent{}, false, nil
	}

	alertName := alert.Labels["alertname"]
	if alertName == "" {
		alertName = "PrometheusAlert"
	}

	message := firstNonEmpty(
		alert.Annotations["summary"],
		alert.Annotations["description"],
		alert.Annotations["message"],
		alertName,
	)

	return model.TimelineEvent{
		Time:         timestamp,
		Source:       "prometheus",
		Category:     "symptom",
		Severity:     normalizeAlertSeverity(alert.Labels["severity"], alert.Status),
		Namespace:    firstNonEmpty(alert.Labels["namespace"], alert.Labels["kubernetes_namespace"]),
		Service:      firstNonEmpty(alert.Labels["service"], alert.Labels["app"], alert.Labels["job"]),
		ResourceKind: "Alert",
		ResourceName: alertName,
		Reason:       alertName,
		Message:      message,
		RiskTags:     tagsForAlert(alertName),
	}, true, nil
}

func normalizeAlertSeverity(severity, status string) string {
	switch strings.ToLower(severity) {
	case "critical", "page", "sev1":
		return "critical"
	case "warning", "sev2":
		return "warning"
	default:
		if strings.EqualFold(status, "resolved") {
			return "info"
		}
		return "warning"
	}
}

func tagsForAlert(alertName string) []string {
	name := strings.ToLower(alertName)
	tags := make([]string, 0, 5)

	if strings.Contains(name, "latency") || strings.Contains(name, "5xx") || strings.Contains(name, "http") || strings.Contains(name, "availability") {
		tags = append(tags, "traffic")
	}
	if strings.Contains(name, "oom") || strings.Contains(name, "memory") {
		tags = append(tags, "memory")
	}
	if strings.Contains(name, "cpu") {
		tags = append(tags, "cpu")
	}
	if strings.Contains(name, "restart") || strings.Contains(name, "crash") {
		tags = append(tags, "restart")
	}
	if strings.Contains(name, "database") || strings.Contains(name, "db") {
		tags = append(tags, "database")
	}

	return dedupe(tags)
}

func parseAlertTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("prometheus alert is missing startsAt")
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse prometheus alert time %q: %w", value, err)
	}
	return timestamp.UTC(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func dedupe(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
