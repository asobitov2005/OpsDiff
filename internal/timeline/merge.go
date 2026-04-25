package timeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func Build(cluster, namespace string, windowStart, windowEnd time.Time, events []model.TimelineEvent, limit int) model.Timeline {
	normalized := NormalizeEvents(events, limit)
	return model.Timeline{
		Version:     "v1",
		Cluster:     cluster,
		Namespace:   namespace,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		GeneratedAt: windowEnd,
		Summary:     SummarizeEvents(normalized),
		Events:      normalized,
	}
}

func Merge(base model.Timeline, extras []model.TimelineEvent, limit int) model.Timeline {
	events := append(append([]model.TimelineEvent{}, base.Events...), extras...)
	base.Events = NormalizeEvents(events, limit)
	base.Summary = SummarizeEvents(base.Events)
	return base
}

func NormalizeEvents(events []model.TimelineEvent, limit int) []model.TimelineEvent {
	normalized := append([]model.TimelineEvent{}, events...)

	sort.Slice(normalized, func(i, j int) bool {
		if !normalized[i].Time.Equal(normalized[j].Time) {
			return normalized[i].Time.Before(normalized[j].Time)
		}
		if normalized[i].Severity != normalized[j].Severity {
			return severityWeight(normalized[i].Severity) < severityWeight(normalized[j].Severity)
		}
		if normalized[i].ResourceKind != normalized[j].ResourceKind {
			return normalized[i].ResourceKind < normalized[j].ResourceKind
		}
		return normalized[i].ResourceName < normalized[j].ResourceName
	})

	if limit > 0 && len(normalized) > limit {
		normalized = normalized[len(normalized)-limit:]
	}

	for index := range normalized {
		normalized[index].ID = fmt.Sprintf("evt_%03d", index+1)
	}

	return normalized
}

func SummarizeEvents(items []model.TimelineEvent) model.TimelineSummary {
	summary := model.TimelineSummary{Total: len(items)}
	for _, item := range items {
		switch item.Severity {
		case "critical":
			summary.Critical++
		case "warning":
			summary.Warning++
		default:
			summary.Info++
		}

		switch item.Category {
		case "change":
			summary.Changes++
		case "symptom":
			summary.Symptoms++
		case "evidence":
			summary.Evidence++
		}

		switch item.Reason {
		case "ContainerRestarted":
			summary.Restarts++
		case "OOMKilled":
			summary.OOMKills++
		case "CrashLoopBackOff":
			summary.CrashLoops++
		}
	}
	return summary
}

func severityWeight(severity string) int {
	switch severity {
	case "info":
		return 1
	case "warning":
		return 2
	case "critical":
		return 3
	default:
		return 0
	}
}
