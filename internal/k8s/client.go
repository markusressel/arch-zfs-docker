package k8s

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/config"
)

// RealClient implements K8sClient talking directly to the Kubernetes API.
type RealClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	namespace  string
	buildNode  string
	repoName   string
}

// NewClient initializes a Kubernetes API client (in-cluster or local kubeconfig).
func NewClient(cfg *config.Config) (K8sClient, error) {
	if cfg.DevMode {
		log.Println("[k8s] Running in dev mode with mock client")
		return NewMockClient(), nil
	}

	// 1. Try In-cluster ServiceAccount
	const (
		tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		caPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	)

	if tokenBytes, err := os.ReadFile(tokenPath); err == nil {
		caBytes, _ := os.ReadFile(caPath)
		client, err := buildHTTPClient(caBytes)
		if err != nil {
			return nil, fmt.Errorf("in-cluster tls config: %w", err)
		}
		log.Println("[k8s] Connected via in-cluster ServiceAccount")
		return &RealClient{
			httpClient: client,
			baseURL:    "https://kubernetes.default.svc",
			token:      strings.TrimSpace(string(tokenBytes)),
			namespace:  cfg.Namespace,
			buildNode:  cfg.BuildNode,
			repoName:   cfg.RepoName,
		}, nil
	}

	// 2. Try Kubeconfig file
	if cfg.Kubeconfig != "" {
		if kc, err := parseKubeconfig(cfg.Kubeconfig); err == nil {
			client, err := buildHTTPClient(kc.caData)
			if err != nil {
				return nil, fmt.Errorf("kubeconfig tls config: %w", err)
			}
			log.Printf("[k8s] Connected via kubeconfig to %s\n", kc.server)
			return &RealClient{
				httpClient: client,
				baseURL:    kc.server,
				token:      kc.token,
				namespace:  cfg.Namespace,
				buildNode:  cfg.BuildNode,
				repoName:   cfg.RepoName,
			}, nil
		}
	}

	log.Println("[k8s] Warning: No Kubernetes credentials found. Falling back to mock client.")
	return NewMockClient(), nil
}

func buildHTTPClient(caData []byte) (*http.Client, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if len(caData) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		tlsConfig.RootCAs = pool
	} else {
		// When developing against local cluster without CA
		tlsConfig.InsecureSkipVerify = true
	}

	return &http.Client{
		Timeout: 0, // No timeout for streaming log connections
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

func (c *RealClient) doReq(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// ListJobs fetches all build jobs in the namespace.
func (c *RealClient) ListJobs() ([]JobSummary, error) {
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs?labelSelector=app.kubernetes.io/name=zfs-repo-builder", c.namespace)
	resp, err := c.doReq("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("k8s api error (%d): %s", resp.StatusCode, string(b))
	}

	var jobList struct {
		Items []struct {
			Metadata struct {
				Name              string    `json:"name"`
				CreationTimestamp time.Time `json:"creationTimestamp"`
			} `json:"metadata"`
			Status struct {
				StartTime      *time.Time `json:"startTime"`
				CompletionTime *time.Time `json:"completionTime"`
				Active         int        `json:"active"`
				Succeeded      int        `json:"succeeded"`
				Failed         int        `json:"failed"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jobList); err != nil {
		return nil, err
	}

	summaries := make([]JobSummary, 0, len(jobList.Items))
	for _, j := range jobList.Items {
		status := "Pending"
		if j.Status.Active > 0 {
			status = "Running"
		} else if j.Status.Succeeded > 0 {
			status = "Succeeded"
		} else if j.Status.Failed > 0 {
			status = "Failed"
		}

		startTime := j.Status.StartTime
		if startTime == nil {
			startTime = &j.Metadata.CreationTimestamp
		}

		var duration int64
		if j.Status.CompletionTime != nil && startTime != nil {
			duration = int64(j.Status.CompletionTime.Sub(*startTime).Seconds())
		} else if startTime != nil {
			duration = int64(time.Since(*startTime).Seconds())
		}

		summaries = append(summaries, JobSummary{
			Name:        j.Metadata.Name,
			Namespace:   c.namespace,
			Status:      status,
			StartTime:   startTime,
			EndTime:     j.Status.CompletionTime,
			DurationSec: duration,
		})
	}

	// Sort newest first
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].StartTime == nil || summaries[j].StartTime == nil {
			return false
		}
		return summaries[i].StartTime.After(*summaries[j].StartTime)
	})

	return summaries, nil
}

// TriggerBuild creates a new Kubernetes Job.
func (c *RealClient) TriggerBuild(req BuildRequest) (*JobSummary, error) {
	jobName := fmt.Sprintf("zfs-build-%d", time.Now().Unix())

	forceBuildStr := "false"
	if req.ForceBuild {
		forceBuildStr = "true"
	}

	envVars := []map[string]string{
		{"name": "REPO_NAME", "value": c.repoName},
		{"name": "VARIANT", "value": req.Variant},
		{"name": "FORCE_BUILD", "value": forceBuildStr},
		{"name": "KERNEL_VERSION", "value": req.KernelVersion},
	}

	jobPayload := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      jobName,
			"namespace": c.namespace,
			"labels": map[string]string{
				"app.kubernetes.io/name":       "zfs-repo-builder",
				"app.kubernetes.io/managed-by": "arch-repo-server",
			},
		},
		"spec": map[string]interface{}{
			"backoffLimit": 2,
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]string{
						"app.kubernetes.io/name": "zfs-repo-builder",
					},
				},
				"spec": map[string]interface{}{
					"restartPolicy": "OnFailure",
					"nodeSelector": map[string]string{
						"kubernetes.io/hostname": c.buildNode,
					},
					"containers": []map[string]interface{}{
						{
							"name":            "builder",
							"image":           "archlinux:base-devel",
							"imagePullPolicy": "IfNotPresent",
							"command":         []string{"/bin/bash", "/scripts/check_and_build.sh"},
							"env":             envVars,
							"volumeMounts": []map[string]string{
								{"name": "repo-data", "mountPath": "/repo"},
								{"name": "builder-scripts", "mountPath": "/scripts"},
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "repo-data",
							"persistentVolumeClaim": map[string]string{
								"claimName": "arch-repo-data",
							},
						},
						{
							"name": "builder-scripts",
							"configMap": map[string]interface{}{
								"name":        "arch-repo-builder-scripts",
								"defaultMode": 0755,
							},
						},
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(jobPayload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", c.namespace)
	resp, err := c.doReq("POST", path, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create job error (%d): %s", resp.StatusCode, string(b))
	}

	now := time.Now()
	return &JobSummary{
		Name:      jobName,
		Namespace: c.namespace,
		Status:    "Pending",
		StartTime: &now,
	}, nil
}

// StreamLogs streams pod logs for a given job as a line-by-line channel.
func (c *RealClient) StreamLogs(jobName string) (<-chan string, error) {
	// 1. Locate pod for the job
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=job-name=%s", c.namespace, jobName)
	resp, err := c.doReq("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("find pod: %w", err)
	}
	defer resp.Body.Close()

	var podList struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&podList); err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pod found for job %s", jobName)
	}

	podName := podList.Items[0].Metadata.Name
	logPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?follow=true", c.namespace, podName)

	logResp, err := c.doReq("GET", logPath, nil)
	if err != nil {
		return nil, fmt.Errorf("stream logs: %w", err)
	}

	if logResp.StatusCode != http.StatusOK {
		defer logResp.Body.Close()
		b, _ := io.ReadAll(logResp.Body)
		return nil, fmt.Errorf("log error (%d): %s", logResp.StatusCode, string(b))
	}

	lines := make(chan string, 100)
	go func() {
		defer logResp.Body.Close()
		defer close(lines)

		scanner := bufio.NewScanner(logResp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	return lines, nil
}
