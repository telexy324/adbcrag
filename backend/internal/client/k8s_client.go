package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type K8sConfig struct {
	APIServer             string
	BearerToken           string
	CACert                string
	InsecureSkipTLSVerify bool
	Timeout               time.Duration
}

type K8sClient interface {
	Test(ctx context.Context, cfg K8sConfig) error
	GetPodContext(ctx context.Context, cfg K8sConfig, namespace, pod, container string, tailLines int, previous bool) (map[string]any, error)
	GetIngressContext(ctx context.Context, cfg K8sConfig, namespace, name string) (map[string]any, error)
	GetServiceContext(ctx context.Context, cfg K8sConfig, namespace, name string) (map[string]any, error)
}

type k8sClient struct{}

func NewK8sClient() K8sClient {
	return &k8sClient{}
}

func (c *k8sClient) Test(ctx context.Context, cfg K8sConfig) error {
	var version map[string]any
	if err := c.getJSON(ctx, cfg, "/version", nil, &version); err != nil {
		return err
	}
	return nil
}

func (c *k8sClient) GetPodContext(ctx context.Context, cfg K8sConfig, namespace, pod, container string, tailLines int, previous bool) (map[string]any, error) {
	result := map[string]any{}
	var podData map[string]any
	if err := c.getJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", url.PathEscape(namespace), url.PathEscape(pod)), nil, &podData); err != nil {
		return nil, err
	}
	result["pod"] = podData
	result["events"] = c.getEvents(ctx, cfg, namespace, "Pod", pod)
	result["currentLogs"] = c.getPodLogs(ctx, cfg, namespace, pod, container, tailLines, false)
	if previous {
		result["previousLogs"] = c.getPodLogs(ctx, cfg, namespace, pod, container, tailLines, true)
	}
	result["ownerWorkloads"] = c.getRelatedWorkloads(ctx, cfg, namespace, podData)
	result["services"] = c.getServicesForPod(ctx, cfg, namespace, podData)
	return result, nil
}

func (c *k8sClient) GetIngressContext(ctx context.Context, cfg K8sConfig, namespace, name string) (map[string]any, error) {
	result := map[string]any{}
	var ingress map[string]any
	if err := c.getJSON(ctx, cfg, fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", url.PathEscape(namespace), url.PathEscape(name)), nil, &ingress); err != nil {
		return nil, err
	}
	result["ingress"] = ingress
	for _, serviceName := range ingressBackendServices(ingress) {
		serviceCtx, err := c.GetServiceContext(ctx, cfg, namespace, serviceName)
		if err == nil {
			result["service:"+serviceName] = serviceCtx
		}
	}
	result["events"] = c.getEvents(ctx, cfg, namespace, "Ingress", name)
	return result, nil
}

func (c *k8sClient) GetServiceContext(ctx context.Context, cfg K8sConfig, namespace, name string) (map[string]any, error) {
	result := map[string]any{}
	var service map[string]any
	if err := c.getJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/services/%s", url.PathEscape(namespace), url.PathEscape(name)), nil, &service); err != nil {
		return nil, err
	}
	result["service"] = service
	result["endpoints"] = c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/endpoints/%s", url.PathEscape(namespace), url.PathEscape(name)), nil)
	result["endpointSlices"] = c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/apis/discovery.k8s.io/v1/namespaces/%s/endpointslices", url.PathEscape(namespace)), url.Values{"labelSelector": []string{"kubernetes.io/service-name=" + name}})
	selector := stringMap(nestedMap(service, "spec", "selector"))
	if len(selector) > 0 {
		result["pods"] = c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/pods", url.PathEscape(namespace)), url.Values{"labelSelector": []string{labelSelector(selector)}})
	}
	result["events"] = c.getEvents(ctx, cfg, namespace, "Service", name)
	return result, nil
}

func (c *k8sClient) getEvents(ctx context.Context, cfg K8sConfig, namespace, kind, name string) any {
	selector := fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", kind, name)
	events := c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/events", url.PathEscape(namespace)), url.Values{"fieldSelector": []string{selector}})
	if items, ok := events.(map[string]any)["items"].([]any); ok && len(items) > 0 {
		return events
	}
	return c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/apis/events.k8s.io/v1/namespaces/%s/events", url.PathEscape(namespace)), url.Values{"fieldSelector": []string{"regarding.kind=" + kind + ",regarding.name=" + name}})
}

func (c *k8sClient) getPodLogs(ctx context.Context, cfg K8sConfig, namespace, pod, container string, tailLines int, previous bool) string {
	if tailLines <= 0 {
		tailLines = 300
	}
	values := url.Values{"tailLines": []string{fmt.Sprint(tailLines)}}
	if container != "" {
		values.Set("container", container)
	}
	if previous {
		values.Set("previous", "true")
	}
	body, err := c.do(ctx, cfg, http.MethodGet, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", url.PathEscape(namespace), url.PathEscape(pod)), values)
	if err != nil {
		return "日志读取失败：" + err.Error()
	}
	return string(body)
}

func (c *k8sClient) getRelatedWorkloads(ctx context.Context, cfg K8sConfig, namespace string, pod map[string]any) any {
	labels := stringMap(nestedMap(pod, "metadata", "labels"))
	if len(labels) == 0 {
		return []any{}
	}
	values := url.Values{"labelSelector": []string{labelSelector(labels)}}
	result := map[string]any{}
	for kind, path := range map[string]string{
		"deployments":  "/apis/apps/v1/namespaces/%s/deployments",
		"statefulsets": "/apis/apps/v1/namespaces/%s/statefulsets",
		"daemonsets":   "/apis/apps/v1/namespaces/%s/daemonsets",
		"replicasets":  "/apis/apps/v1/namespaces/%s/replicasets",
	} {
		result[kind] = c.getOptionalJSON(ctx, cfg, fmt.Sprintf(path, url.PathEscape(namespace)), values)
	}
	return result
}

func (c *k8sClient) getServicesForPod(ctx context.Context, cfg K8sConfig, namespace string, pod map[string]any) any {
	podLabels := stringMap(nestedMap(pod, "metadata", "labels"))
	services := c.getOptionalJSON(ctx, cfg, fmt.Sprintf("/api/v1/namespaces/%s/services", url.PathEscape(namespace)), nil)
	items, _ := services.(map[string]any)["items"].([]any)
	matched := []any{}
	for _, item := range items {
		svc, _ := item.(map[string]any)
		selector := stringMap(nestedMap(svc, "spec", "selector"))
		if len(selector) > 0 && labelsMatch(podLabels, selector) {
			matched = append(matched, svc)
		}
	}
	return matched
}

func (c *k8sClient) getOptionalJSON(ctx context.Context, cfg K8sConfig, path string, values url.Values) any {
	var data map[string]any
	if err := c.getJSON(ctx, cfg, path, values, &data); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return data
}

func (c *k8sClient) getJSON(ctx context.Context, cfg K8sConfig, path string, values url.Values, target any) error {
	body, err := c.do(ctx, cfg, http.MethodGet, path, values)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func (c *k8sClient) do(ctx context.Context, cfg K8sConfig, method, path string, values url.Values) ([]byte, error) {
	endpoint := strings.TrimRight(cfg.APIServer, "/") + path
	if values != nil && len(values) > 0 {
		endpoint += "?" + values.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.BearerToken)
	}
	httpClient, err := newK8sHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kubernetes api %s returned %s: %s", path, resp.Status, string(body))
	}
	return body, nil
}

func newK8sHTTPClient(cfg K8sConfig) (*http.Client, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipTLSVerify} //nolint:gosec
	if cfg.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.CACert)) {
			return nil, fmt.Errorf("invalid caCert")
		}
		tlsConfig.RootCAs = pool
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &http.Client{Timeout: timeout, Transport: &http.Transport{TLSClientConfig: tlsConfig}}, nil
}

func nestedMap(root map[string]any, keys ...string) map[string]any {
	current := root
	for _, key := range keys {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func stringMap(input map[string]any) map[string]string {
	out := map[string]string{}
	for key, value := range input {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
}

func labelSelector(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for key, value := range labels {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func labelsMatch(labels, selector map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func ingressBackendServices(ingress map[string]any) []string {
	names := map[string]bool{}
	spec := nestedMap(ingress, "spec")
	if backend := nestedMap(spec, "defaultBackend", "service"); backend != nil {
		if name, _ := backend["name"].(string); name != "" {
			names[name] = true
		}
	}
	rules, _ := spec["rules"].([]any)
	for _, rule := range rules {
		ruleMap, _ := rule.(map[string]any)
		paths, _ := nestedMap(ruleMap, "http")["paths"].([]any)
		for _, path := range paths {
			pathMap, _ := path.(map[string]any)
			service := nestedMap(pathMap, "backend", "service")
			if name, _ := service["name"].(string); name != "" {
				names[name] = true
			}
		}
	}
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	return out
}
