package argocd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

type applicationEnvelope struct {
	Applications []applicationPayload `json:"applications"`
}

type applicationPayload struct {
	App                  string `json:"app"`
	Name                 string `json:"name"`
	Application          string `json:"application"`
	Time                 string `json:"time"`
	StartedAt            string `json:"startedAt"`
	FinishedAt           string `json:"finishedAt"`
	Revision             string `json:"revision"`
	SyncStatus           string `json:"syncStatus"`
	HealthStatus         string `json:"healthStatus"`
	OperationPhase       string `json:"operationPhase"`
	DestinationNamespace string `json:"destinationNamespace"`
	Namespace            string `json:"namespace"`
}

func LoadTimelineEvents(path string, windowStart, windowEnd time.Time) ([]model.TimelineEvent, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read argocd events %q: %w", path, err)
	}

	var applications []applicationPayload
	if err := json.Unmarshal(data, &applications); err != nil {
		var envelope applicationEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, fmt.Errorf("decode argocd events %q: %w", path, err)
		}
		applications = envelope.Applications
	}

	events := make([]model.TimelineEvent, 0, len(applications))
	for _, application := range applications {
		event, ok, err := normalizeApplication(application, windowStart, windowEnd)
		if err != nil {
			return nil, err
		}
		if ok {
			events = append(events, event)
		}
	}

	return events, nil
}

func normalizeApplication(application applicationPayload, windowStart, windowEnd time.Time) (model.TimelineEvent, bool, error) {
	timestamp, err := parseApplicationTime(application)
	if err != nil {
		return model.TimelineEvent{}, false, err
	}
	if timestamp.Before(windowStart) || timestamp.After(windowEnd) {
		return model.TimelineEvent{}, false, nil
	}

	appName := firstNonEmpty(application.App, application.Name, application.Application)
	if appName == "" {
		appName = "argocd-app"
	}

	severity := "info"
	if strings.EqualFold(application.OperationPhase, "failed") || strings.EqualFold(application.HealthStatus, "degraded") {
		severity = "warning"
	}

	message := fmt.Sprintf("ArgoCD synced %s", appName)
	if application.Revision != "" {
		message = fmt.Sprintf("%s to revision %s", message, application.Revision)
	}
	if application.SyncStatus != "" {
		message = fmt.Sprintf("%s (%s)", message, application.SyncStatus)
	}

	reason := "SyncObserved"
	if application.OperationPhase != "" {
		reason = "Sync" + upperFirst(strings.ToLower(application.OperationPhase))
	}

	return model.TimelineEvent{
		Time:         timestamp,
		Source:       "argocd",
		Category:     "change",
		Severity:     severity,
		Namespace:    firstNonEmpty(application.DestinationNamespace, application.Namespace),
		Service:      appName,
		ResourceKind: "Application",
		ResourceName: appName,
		Reason:       reason,
		Message:      message,
		RiskTags:     []string{"rollout-evidence"},
	}, true, nil
}

func parseApplicationTime(application applicationPayload) (time.Time, error) {
	value := firstNonEmpty(application.Time, application.FinishedAt, application.StartedAt)
	if value == "" {
		return time.Time{}, fmt.Errorf("argocd application event is missing time fields")
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse argocd time %q: %w", value, err)
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

func upperFirst(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
