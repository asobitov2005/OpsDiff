package explain

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/diff"
	"github.com/asobitov2005/OpsDiff/internal/model"
)

type Engine struct{}

func NewEngine() Engine {
	return Engine{}
}

func (Engine) Explain(before, after model.Snapshot, timeline model.Timeline, beforePath, afterPath string, top int) model.ExplainResult {
	compare := diff.NewEngine().Compare(before, after, beforePath, afterPath)
	candidates := make([]model.ExplainCandidate, 0, len(compare.Changes))

	for _, change := range compare.Changes {
		candidate := scoreChange(change, before, after, timeline)
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Change.Risk != candidates[j].Change.Risk {
			return riskWeight(candidates[i].Change.Risk) > riskWeight(candidates[j].Change.Risk)
		}
		if candidates[i].Service != candidates[j].Service {
			return candidates[i].Service < candidates[j].Service
		}
		return candidates[i].Change.Path < candidates[j].Change.Path
	})

	if top > 0 && len(candidates) > top {
		candidates = candidates[:top]
	}

	for index := range candidates {
		candidates[index].Rank = index + 1
	}

	result := model.ExplainResult{
		BeforePath:      beforePath,
		AfterPath:       afterPath,
		Namespace:       after.Namespace,
		GeneratedAt:     time.Now().UTC(),
		ChangeWindow:    model.TimeWindow{Start: before.CreatedAt, End: after.CreatedAt},
		CompareSummary:  compare.Summary,
		TimelineSummary: timeline.Summary,
		Candidates:      candidates,
	}
	result.Summary = summarizeExplain(timeline, candidates, compare.Summary.Total)

	return result
}

func scoreChange(change model.Change, before, after model.Snapshot, timeline model.Timeline) model.ExplainCandidate {
	service := inferChangeService(change)
	baseScore := baseRiskScore(change.Risk)
	evidence := []string{fmt.Sprintf("%s risk change: %s", strings.ToUpper(string(change.Risk)), change.Summary)}

	matches := make([]scoredEvent, 0)
	changeTags := tagsForChange(change)
	for _, event := range timeline.Events {
		match := scoreEvent(change, service, changeTags, before.CreatedAt, after.CreatedAt, event)
		if match.Contribution > 0 {
			matches = append(matches, match)
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Contribution != matches[j].Contribution {
			return matches[i].Contribution > matches[j].Contribution
		}
		return matches[i].Event.Time.After(matches[j].Event.Time)
	})

	score := baseScore
	matchedEvents := make([]model.ExplainEventMatch, 0, 3)
	for index, match := range matches {
		if index >= 3 {
			break
		}
		weighted := weightedContribution(match.Contribution, index)
		score += weighted
		evidence = append(evidence, match.Reason)
		matchedEvents = append(matchedEvents, model.ExplainEventMatch{
			Time:         match.Event.Time,
			Severity:     match.Event.Severity,
			Category:     match.Event.Category,
			Reason:       match.Event.Reason,
			Message:      match.Event.Message,
			ResourceKind: match.Event.ResourceKind,
			ResourceName: match.Event.ResourceName,
			Contribution: weighted,
		})
	}

	if countCategory(matches, "symptom") >= 2 {
		score += 8
		evidence = append(evidence, "multiple runtime symptoms align with this change")
	}
	if countCategory(matches, "evidence") >= 1 {
		score += 5
		evidence = append(evidence, "runtime evidence supports this candidate")
	}

	if score > 100 {
		score = 100
	}

	return model.ExplainCandidate{
		Score:           score,
		Likelihood:      likelihoodLabel(score),
		Service:         service,
		Change:          change,
		Evidence:        dedupeStrings(evidence),
		SuggestedChecks: suggestChecks(changeTags, matches),
		MatchedEvents:   matchedEvents,
	}
}

type scoredEvent struct {
	Event        model.TimelineEvent
	Contribution int
	Category     string
	Reason       string
}

func scoreEvent(change model.Change, service string, changeTags []string, windowStart, windowEnd time.Time, event model.TimelineEvent) scoredEvent {
	if event.Namespace != "" && change.Namespace != "" && event.Namespace != change.Namespace {
		return scoredEvent{}
	}

	score := 0
	reasons := make([]string, 0, 4)
	eventTags := tagsForEvent(event)

	if service != "" && event.Service == service {
		score += 22
		reasons = append(reasons, fmt.Sprintf("same service `%s` showed runtime activity", service))
	}

	if event.ResourceName == change.ResourceName && event.ResourceKind == change.ResourceKind {
		score += 14
		reasons = append(reasons, fmt.Sprintf("same resource `%s/%s` appears in the incident timeline", change.ResourceKind, change.ResourceName))
	} else if event.ResourceName != "" && strings.Contains(event.ResourceName, service) && service != "" {
		score += 8
		reasons = append(reasons, "timeline resource name aligns with the changed service")
	}

	overlap := sharedTags(changeTags, eventTags)
	if len(overlap) > 0 {
		overlapScore := 18 + len(overlap)*4
		score += overlapScore
		reasons = append(reasons, fmt.Sprintf("symptom mapping matched on `%s`", strings.Join(overlap, ", ")))
	}

	score += severityScore(event)
	if extra, ok := specializedBoost(changeTags, event); ok {
		score += extra
		reasons = append(reasons, specializedReason(event))
	}

	if bonus, why := timeProximityScore(windowStart, windowEnd, event.Time); bonus > 0 {
		score += bonus
		reasons = append(reasons, why)
	}

	if event.Category == "change" && hasTag(event.RiskTags, "rollout-evidence") && service != "" && event.Service == service {
		score += 8
		reasons = append(reasons, "rollout evidence was observed for the same service")
	}

	if score == 0 {
		return scoredEvent{}
	}

	return scoredEvent{
		Event:        event,
		Contribution: score,
		Category:     event.Category,
		Reason:       strings.Join(dedupeStrings(reasons), "; "),
	}
}

func baseRiskScore(risk model.Risk) int {
	switch risk {
	case model.RiskHigh:
		return 35
	case model.RiskMedium:
		return 22
	default:
		return 10
	}
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

func inferChangeService(change model.Change) string {
	name := change.ResourceName
	suffixes := []string{
		"-config",
		"-configmap",
		"-secret",
		"-svc",
		"-service",
		"-ingress",
		"-hpa",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func tagsForChange(change model.Change) []string {
	tags := make([]string, 0, 6)
	path := strings.ToLower(change.Path)
	name := strings.ToLower(change.ResourceName + "." + change.Path)

	if strings.Contains(path, "memory") {
		tags = append(tags, "memory")
	}
	if strings.Contains(path, "cpu") {
		tags = append(tags, "cpu")
	}
	if strings.Contains(path, ".image") {
		tags = append(tags, "image")
	}
	if strings.Contains(path, ".env.") {
		tags = append(tags, "env")
	}
	if change.ResourceKind == "ConfigMap" {
		tags = append(tags, "config")
	}
	if change.ResourceKind == "Secret" {
		tags = append(tags, "secret")
	}
	if change.ResourceKind == "Service" || change.ResourceKind == "Ingress" || strings.Contains(path, "selector") {
		tags = append(tags, "traffic")
	}
	if change.ResourceKind == "HorizontalPodAutoscaler" || strings.Contains(path, "replicas") {
		tags = append(tags, "scaling")
	}
	if strings.Contains(name, "db_") || strings.Contains(name, "database") || strings.Contains(name, "pool") || strings.Contains(name, "connection") {
		tags = append(tags, "database")
	}
	if strings.Contains(name, "timeout") || strings.Contains(name, "probe") {
		tags = append(tags, "probe")
	}

	return dedupeStrings(tags)
}

func tagsForEvent(event model.TimelineEvent) []string {
	tags := append([]string{}, event.RiskTags...)
	switch event.Reason {
	case "OOMKilled":
		tags = append(tags, "memory", "image", "env", "config")
	case "CrashLoopBackOff", "BackOff":
		tags = append(tags, "image", "env", "config", "secret")
	case "ContainerRestarted":
		tags = append(tags, "restart")
	case "FailedMount":
		tags = append(tags, "secret", "config", "storage")
	case "FailedScheduling":
		tags = append(tags, "cpu", "memory", "scaling")
	case "Unhealthy":
		tags = append(tags, "probe", "traffic", "env", "config")
	case "ScalingReplicaSet":
		tags = append(tags, "rollout", "scaling")
	}

	return dedupeStrings(tags)
}

func sharedTags(changeTags, eventTags []string) []string {
	eventSet := make(map[string]struct{}, len(eventTags))
	for _, tag := range eventTags {
		eventSet[tag] = struct{}{}
	}

	out := make([]string, 0)
	for _, tag := range changeTags {
		if _, ok := eventSet[tag]; ok {
			out = append(out, tag)
		}
	}
	return dedupeStrings(out)
}

func severityScore(event model.TimelineEvent) int {
	switch event.Severity {
	case "critical":
		return 16
	case "warning":
		return 9
	default:
		return 3
	}
}

func specializedBoost(changeTags []string, event model.TimelineEvent) (int, bool) {
	switch event.Reason {
	case "OOMKilled":
		if containsTag(changeTags, "memory") {
			return 24, true
		}
	case "CrashLoopBackOff", "BackOff":
		if containsAnyTag(changeTags, "image", "env", "config", "secret") {
			return 20, true
		}
	case "FailedMount":
		if containsAnyTag(changeTags, "config", "secret") {
			return 18, true
		}
	case "FailedScheduling":
		if containsAnyTag(changeTags, "cpu", "memory", "scaling") {
			return 18, true
		}
	case "Unhealthy":
		if containsAnyTag(changeTags, "probe", "traffic", "config", "env") {
			return 16, true
		}
	}
	return 0, false
}

func specializedReason(event model.TimelineEvent) string {
	switch event.Reason {
	case "OOMKilled":
		return "memory-related runtime failure directly matches the changed field"
	case "CrashLoopBackOff", "BackOff":
		return "crash-loop symptoms match common config, secret, env, or image regressions"
	case "FailedMount":
		return "mount failures often follow config or secret regressions"
	case "FailedScheduling":
		return "scheduling failures align with CPU, memory, or scaling changes"
	case "Unhealthy":
		return "probe failures align with traffic, config, env, or probe-related changes"
	default:
		return "runtime symptom mapping matched this change"
	}
}

func timeProximityScore(windowStart, windowEnd, eventTime time.Time) (int, string) {
	if eventTime.IsZero() {
		return 0, ""
	}
	switch {
	case !eventTime.Before(windowStart) && !eventTime.After(windowEnd):
		return 20, "symptom appeared inside the snapshot change window"
	case eventTime.After(windowEnd):
		delta := eventTime.Sub(windowEnd)
		switch {
		case delta <= 15*time.Minute:
			return 24, "symptom appeared within 15 minutes after the changed snapshot"
		case delta <= time.Hour:
			return 14, "symptom appeared within 1 hour after the changed snapshot"
		default:
			return 6, "symptom appeared after the changed snapshot window"
		}
	case windowStart.Sub(eventTime) <= 15*time.Minute:
		return 6, "runtime pressure started shortly before the changed snapshot window"
	default:
		return 0, ""
	}
}

func likelihoodLabel(score int) string {
	switch {
	case score >= 85:
		return "high"
	case score >= 60:
		return "medium"
	default:
		return "low"
	}
}

func summarizeExplain(timeline model.Timeline, candidates []model.ExplainCandidate, totalChanges int) model.ExplainSummary {
	summary := model.ExplainSummary{
		TotalChanges:       totalChanges,
		RankedCandidates:   len(candidates),
		CriticalSymptoms:   timeline.Summary.Critical,
		WarningSymptoms:    timeline.Summary.Warning,
		SupportingEvidence: timeline.Summary.Evidence,
	}
	return summary
}

func suggestChecks(changeTags []string, matches []scoredEvent) []string {
	checks := make([]string, 0, 4)

	if containsTag(changeTags, "memory") {
		checks = append(checks, "Inspect pod memory usage and recent OOMKilled events")
	}
	if containsTag(changeTags, "image") {
		checks = append(checks, "Compare container startup behavior and consider rolling back the image")
	}
	if containsAnyTag(changeTags, "config", "env", "secret", "database") {
		checks = append(checks, "Verify configuration values, dependent secrets, and downstream connectivity")
	}
	if containsAnyTag(changeTags, "traffic", "probe") {
		checks = append(checks, "Validate readiness paths, service selectors, and ingress/service routing")
	}
	if containsTag(changeTags, "scaling") {
		checks = append(checks, "Check replica counts, HPA decisions, and scheduling pressure")
	}

	for _, match := range matches {
		if match.Event.Reason == "FailedMount" {
			checks = append(checks, "Confirm referenced ConfigMaps and Secrets exist and are mounted correctly")
		}
	}

	return dedupeStrings(checks)
}

func countCategory(matches []scoredEvent, category string) int {
	total := 0
	for _, match := range matches {
		if match.Category == category {
			total++
		}
	}
	return total
}

func containsAnyTag(tags []string, targets ...string) bool {
	for _, target := range targets {
		if containsTag(tags, target) {
			return true
		}
	}
	return false
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func hasTag(tags []string, target string) bool {
	return containsTag(tags, target)
}

func dedupeStrings(items []string) []string {
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

func weightedContribution(value, index int) int {
	switch index {
	case 0:
		return value / 2
	case 1:
		return value / 4
	default:
		return value / 6
	}
}
