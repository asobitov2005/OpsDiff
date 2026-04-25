package kube

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNormalizeClusterEventsFiltersAndMapsRelevantItems(t *testing.T) {
	windowStart := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)

	items := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "backoff",
				Namespace:         "prod",
				CreationTimestamp: metav1.NewTime(windowStart.Add(5 * time.Minute)),
			},
			Type:    corev1.EventTypeWarning,
			Reason:  "BackOff",
			Message: "Back-off restarting failed container",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "api-7b6d9c8f6f-x2z9w",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "rollout",
				Namespace:         "prod",
				CreationTimestamp: metav1.NewTime(windowStart.Add(3 * time.Minute)),
			},
			Type:    corev1.EventTypeNormal,
			Reason:  "ScalingReplicaSet",
			Message: "Scaled up replica set api-7b6d9c8f6f to 3",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Deployment",
				Name: "api",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ignored",
				Namespace:         "prod",
				CreationTimestamp: metav1.NewTime(windowStart.Add(4 * time.Minute)),
			},
			Type:    corev1.EventTypeNormal,
			Reason:  "Started",
			Message: "Started container api",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "api-7b6d9c8f6f-x2z9w",
			},
		},
	}

	events := normalizeClusterEvents(items, windowStart)
	if len(events) != 2 {
		t.Fatalf("expected 2 normalized events, got %d", len(events))
	}

	if events[0].Reason != "BackOff" || events[0].Severity != "critical" || events[0].Category != "symptom" {
		t.Fatalf("unexpected warning event mapping: %+v", events[0])
	}

	if events[1].Reason != "ScalingReplicaSet" || events[1].Category != "change" || events[1].Severity != "info" {
		t.Fatalf("unexpected rollout event mapping: %+v", events[1])
	}
}

func TestPodSignalEventsDetectRestartOOMAndCrashLoop(t *testing.T) {
	windowStart := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	observedAt := windowStart.Add(10 * time.Minute)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "api-7b6d9c8f6f-x2z9w",
			Namespace:         "prod",
			CreationTimestamp: metav1.NewTime(windowStart.Add(2 * time.Minute)),
			Labels: map[string]string{
				"app.kubernetes.io/name": "api",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "api",
					RestartCount: 4,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:     "OOMKilled",
							FinishedAt: metav1.NewTime(windowStart.Add(8 * time.Minute)),
						},
					},
				},
			},
		},
	}

	events := podSignalEvents([]corev1.Pod{pod}, windowStart, observedAt)
	if len(events) != 3 {
		t.Fatalf("expected 3 pod signal events, got %d", len(events))
	}

	seen := map[string]bool{}
	for _, event := range events {
		seen[event.Reason] = true
		if event.Service != "api" {
			t.Fatalf("expected service inference to be api, got %+v", event)
		}
	}

	for _, reason := range []string{"ContainerRestarted", "OOMKilled", "CrashLoopBackOff"} {
		if !seen[reason] {
			t.Fatalf("missing signal event %s", reason)
		}
	}
}
