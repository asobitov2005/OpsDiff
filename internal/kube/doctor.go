package kube

import (
	"context"
	"fmt"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail,omitempty"`
	FixHint string `json:"fix_hint,omitempty"`
}

func RunDoctor(ctx context.Context, kubeconfigPath, namespace string) []Check {
	checks := make([]Check, 0, 8)
	resolvedPath := ResolveKubeconfigPath(kubeconfigPath)

	if _, err := os.Stat(resolvedPath); err != nil {
		return append(checks, Check{
			Name:    "kubeconfig found",
			Passed:  false,
			Detail:  err.Error(),
			FixHint: "set --kubeconfig or ensure ~/.kube/config exists",
		})
	}

	checks = append(checks, Check{
		Name:   "kubeconfig found",
		Passed: true,
		Detail: resolvedPath,
	})

	restConfig, rawConfig, _, err := LoadRESTConfig(kubeconfigPath)
	if err != nil {
		return append(checks, Check{
			Name:    "kubeconfig is valid",
			Passed:  false,
			Detail:  err.Error(),
			FixHint: "fix the kubeconfig context, user credentials, or cluster endpoint",
		})
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return append(checks, Check{
			Name:    "kubernetes client created",
			Passed:  false,
			Detail:  err.Error(),
			FixHint: "verify the kubeconfig credentials and cluster certificate settings",
		})
	}

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return append(checks, Check{
			Name:    "connected to cluster",
			Passed:  false,
			Detail:  err.Error(),
			FixHint: "check API server reachability and credentials",
		})
	}

	clusterName := CurrentClusterName(rawConfig)
	if clusterName == "" {
		clusterName = "unknown"
	}

	checks = append(checks, Check{
		Name:   "connected to cluster",
		Passed: true,
		Detail: fmt.Sprintf("%s (%s)", clusterName, version.GitVersion),
	})

	scope := namespace
	if scope == "" {
		scope = metav1.NamespaceAll
	}

	scopeLabel := namespace
	if scopeLabel == "" {
		scopeLabel = "all namespaces"
	}

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list deployments in %s", scopeLabel),
		func() error {
			_, err := clientset.AppsV1().Deployments(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for deployments.apps in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list configmaps in %s", scopeLabel),
		func() error {
			_, err := clientset.CoreV1().ConfigMaps(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for configmaps in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list secrets in %s", scopeLabel),
		func() error {
			_, err := clientset.CoreV1().Secrets(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for secrets in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list services in %s", scopeLabel),
		func() error {
			_, err := clientset.CoreV1().Services(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for services in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list pods in %s", scopeLabel),
		func() error {
			_, err := clientset.CoreV1().Pods(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for pods in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list events in %s", scopeLabel),
		func() error {
			_, err := clientset.CoreV1().Events(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for events in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list ingresses in %s", scopeLabel),
		func() error {
			_, err := clientset.NetworkingV1().Ingresses(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for ingresses.networking.k8s.io in %s", scopeLabel),
	))

	checks = append(checks, listCheck(
		ctx,
		fmt.Sprintf("can list HPAs in %s", scopeLabel),
		func() error {
			_, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(scope).List(ctx, metav1.ListOptions{Limit: 1})
			return err
		},
		fmt.Sprintf("grant list permission for horizontalpodautoscalers.autoscaling in %s", scopeLabel),
	))

	return checks
}

func HasFailures(checks []Check) bool {
	for _, check := range checks {
		if !check.Passed {
			return true
		}
	}
	return false
}

func listCheck(_ context.Context, name string, action func() error, fixHint string) Check {
	if err := action(); err != nil {
		detail := err.Error()
		if apierrors.IsForbidden(err) {
			detail = "permission denied"
		}

		return Check{
			Name:    name,
			Passed:  false,
			Detail:  detail,
			FixHint: fixHint,
		}
	}

	return Check{
		Name:   name,
		Passed: true,
	}
}
