package k8s

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/config"
	"github.com/markusressel/arch-zfs-docker/internal/kernel"
)

// RealClient implements K8sClient talking directly to the Kubernetes API.
type RealClient struct {
	mu sync.RWMutex
	// builderImage is the image build jobs copy the builder from; resolved on first use when empty.
	builderImage string
	httpClient   *http.Client
	baseURL      string
	token        string
	namespace    string
	buildNode    string
	repoName     string
	repoDir      string
	// inCluster is set when the service runs in a pod and can look up its own image.
	inCluster bool
}

// SetBuildNode dynamically updates the target build node.
func (c *RealClient) SetBuildNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buildNode = node
}

func (c *RealClient) getBuildNode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buildNode
}

// NewClient initializes a Kubernetes API client (in-cluster or local kubeconfig).
func NewClient(cfg *config.Config) (K8sClient, error) {
	if cfg.DevMode {
		log.Println("[k8s] Running in dev mode with mock client")
		cfg.SetActiveRunner("mock")
		return NewMockClient(), nil
	}

	runner := cfg.BuildRunner
	switch runner {
	case "", "auto", "kubernetes", "docker":
	default:
		return nil, fmt.Errorf("unknown build runner %q (use auto, kubernetes or docker)", runner)
	}

	if runner != "docker" {
		client, err := newKubernetesClient(cfg)
		if err != nil {
			return nil, err
		}
		if client != nil {
			cfg.SetActiveRunner("kubernetes")
			return client, nil
		}
		if runner == "kubernetes" {
			return nil, fmt.Errorf("build runner kubernetes requested, but no Kubernetes credentials were found")
		}
	}

	dockerClient, err := newDockerRunner(cfg)
	if err == nil {
		log.Printf("[docker] No Kubernetes available, running builds in the local Docker engine (mounting %s at /repo)\n", dockerClient.repoMount)
		cfg.SetActiveRunner("docker")
		return dockerClient, nil
	}
	if runner == "docker" {
		return nil, err
	}
	log.Printf("[k8s] Warning: no Kubernetes credentials and %v. Falling back to mock client.\n", err)
	cfg.SetActiveRunner("mock")
	return NewMockClient(), nil
}

// newKubernetesClient returns nil (and no error) when no cluster credentials are available.
func newKubernetesClient(cfg *config.Config) (K8sClient, error) {
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
			httpClient:   client,
			baseURL:      "https://kubernetes.default.svc",
			token:        strings.TrimSpace(string(tokenBytes)),
			namespace:    cfg.Namespace,
			buildNode:    cfg.BuildNode,
			repoName:     cfg.RepoName,
			repoDir:      cfg.RepoDir,
			builderImage: cfg.BuilderImage,
			inCluster:    true,
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
				httpClient:   client,
				baseURL:      kc.server,
				token:        kc.token,
				namespace:    cfg.Namespace,
				buildNode:    cfg.BuildNode,
				repoName:     cfg.RepoName,
				repoDir:      cfg.RepoDir,
				builderImage: cfg.BuilderImage,
			}, nil
		}
	}
	return nil, nil
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

func (c *RealClient) doReq(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
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

const (
	annotationVariant = "zfs-repo-builder/variant"
	annotationKernel  = "zfs-repo-builder/kernel-version"
	annotationZfs     = "zfs-repo-builder/zfs-version"
	annotationTask    = "zfs-repo-builder/task"
	annotationPackage = "zfs-repo-builder/package"
	// "true" when the build may target a kernel the OpenZFS release does not support.
	annotationUnsupported = "zfs-repo-builder/unsupported-kernel"
)

// DeleteJob removes a build job and its pods from the cluster.
func (c *RealClient) DeleteJob(name string) error {
	if !ValidJobName(name) {
		return fmt.Errorf("invalid job name %q", name)
	}
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s?propagationPolicy=Background", c.namespace, name)
	resp, err := c.doReq(context.Background(), "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrJobNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete job error (%d): %s", resp.StatusCode, string(b))
	}
	return nil
}

// kubernetesJobStatus maps the pod counters of a Job's status to a JobStatus. Failed pods
// only make the job failed once nothing runs anymore (the job retries up to its backoffLimit).
func kubernetesJobStatus(active, succeeded, failed int) JobStatus {
	switch {
	case active > 0:
		return StatusRunning
	case succeeded > 0:
		return StatusSucceeded
	case failed > 0:
		return StatusFailed
	default:
		return StatusPending
	}
}

// ListJobs fetches all build jobs in the namespace.
func (c *RealClient) ListJobs() ([]JobSummary, error) {
	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs?labelSelector=app.kubernetes.io/name=zfs-repo-builder", c.namespace)
	resp, err := c.doReq(context.Background(), "GET", path, nil)
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
				Name              string            `json:"name"`
				CreationTimestamp time.Time         `json:"creationTimestamp"`
				Annotations       map[string]string `json:"annotations"`
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
		status := kubernetesJobStatus(j.Status.Active, j.Status.Succeeded, j.Status.Failed)

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
			Name:              j.Metadata.Name,
			Namespace:         c.namespace,
			Status:            status,
			StartTime:         startTime,
			EndTime:           j.Status.CompletionTime,
			DurationSec:       duration,
			Variant:           j.Metadata.Annotations[annotationVariant],
			KernelVersion:     j.Metadata.Annotations[annotationKernel],
			ZfsVersion:        j.Metadata.Annotations[annotationZfs],
			Task:              j.Metadata.Annotations[annotationTask],
			Package:           j.Metadata.Annotations[annotationPackage],
			UnsupportedKernel: j.Metadata.Annotations[annotationUnsupported] == "true",
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
	jobName := newJobName()
	req.KernelVersion = kernel.Normalize(req.KernelVersion)

	envVars := buildEnv(jobName, c.repoName, req)

	podSpec := map[string]interface{}{
		"restartPolicy": "OnFailure",
		"initContainers": []map[string]interface{}{
			{
				// The service's own image provides the builder binary, so job and service always match.
				"name":    "install-scripts",
				"image":   c.jobBuilderImage(context.Background()),
				"command": []string{"/bin/sh", "-c"},
				"args": []string{
					"cp " + serverBinaryInImage + " " + builderPath,
				},
				"volumeMounts": []map[string]string{
					{"name": "builder-scripts", "mountPath": "/scripts"},
				},
			},
		},
		"containers": []map[string]interface{}{
			{
				"name":            "builder",
				"image":           "archlinux:base-devel",
				"imagePullPolicy": "IfNotPresent",
				"command":         jobCommand(req),
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
				"name":     "builder-scripts",
				"emptyDir": map[string]interface{}{},
			},
		},
	}

	if node := c.getBuildNode(); node != "" {
		podSpec["nodeSelector"] = map[string]string{
			"kubernetes.io/hostname": node,
		}
	}

	jobPayload := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      jobName,
			"namespace": c.namespace,
			"annotations": map[string]string{
				annotationVariant:     req.Variant,
				annotationKernel:      req.KernelVersion,
				annotationZfs:         req.ZfsVersion,
				annotationTask:        req.Task,
				annotationPackage:     req.Package,
				annotationUnsupported: strconv.FormatBool(req.AllowUnsupportedKernel),
			},
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
				"spec": podSpec,
			},
		},
	}

	payloadBytes, err := json.Marshal(jobPayload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", c.namespace)
	resp, err := c.doReq(context.Background(), "POST", path, bytes.NewReader(payloadBytes))
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
		Name:              jobName,
		Namespace:         c.namespace,
		Status:            StatusPending,
		StartTime:         &now,
		Variant:           req.Variant,
		KernelVersion:     req.KernelVersion,
		ZfsVersion:        req.ZfsVersion,
		Task:              req.Task,
		Package:           req.Package,
		UnsupportedKernel: req.AllowUnsupportedKernel,
	}, nil
}

// Pod phases in which a pod's log can be followed.
const (
	podPending = "Pending"
	podRunning = "Running"
)

// findPod returns the pod of a job and its phase, or an empty name if there is none (yet).
func (c *RealClient) findPod(ctx context.Context, jobName string) (name, phase string, err error) {
	q := url.Values{"labelSelector": {"job-name=" + jobName}}
	resp, err := c.doReq(ctx, "GET", fmt.Sprintf("/api/v1/namespaces/%s/pods?%s", c.namespace, q.Encode()), nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("list pods (%d)", resp.StatusCode)
	}
	var podList struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&podList); err != nil {
		return "", "", err
	}
	if len(podList.Items) == 0 {
		return "", "", nil
	}
	return podList.Items[0].Metadata.Name, podList.Items[0].Status.Phase, nil
}

// StreamLogs follows the log of a job's pod, or replays the persisted log file of a finished build.
func (c *RealClient) StreamLogs(ctx context.Context, jobName string) (<-chan string, error) {
	// The name ends up in a file path and in API queries.
	if !ValidJobName(jobName) {
		return nil, ErrJobNotFound
	}
	logFilePath := filepath.Join(c.repoDir, "logs", jobName+".log")

	// A finished build's pod may be gone already; its log file is complete.
	if _, err := os.Stat(logFilePath); err == nil {
		if _, phase, err := c.findPod(ctx, jobName); err != nil || (phase != podRunning && phase != podPending) {
			return streamFromFile(ctx, logFilePath)
		}
	}

	// Wait for the pod of a just created job (or for its log file to show up).
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var podName string
	for {
		if name, _, err := c.findPod(ctx, jobName); err == nil && name != "" {
			podName = name
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			if fileLines, fileErr := streamFromFile(ctx, logFilePath); fileErr == nil {
				return fileLines, nil
			}
			return nil, fmt.Errorf("timeout waiting for pod of job %s to be created", jobName)
		case <-ticker.C:
			if fileLines, fileErr := streamFromFile(ctx, logFilePath); fileErr == nil {
				return fileLines, nil
			}
		}
	}

	logPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log?follow=true", c.namespace, url.PathEscape(podName))
	logResp, err := c.doReq(ctx, "GET", logPath, nil)
	if err == nil && logResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(logResp.Body, 4096))
		logResp.Body.Close()
		err = fmt.Errorf("log error (%d): %s", logResp.StatusCode, string(b))
	}
	if err != nil {
		if fileLines, fileErr := streamFromFile(ctx, logFilePath); fileErr == nil {
			return fileLines, nil
		}
		return nil, fmt.Errorf("stream logs: %w", err)
	}
	return streamLines(ctx, logResp.Body), nil
}

// jobBuilderImage returns the image build jobs copy the builder binary from: the configured one, else
// the image of the pod this service runs in, else the published image.
func (c *RealClient) jobBuilderImage(ctx context.Context) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.builderImage != "" {
		return c.builderImage
	}
	image, err := c.ownImage(ctx)
	if err != nil {
		log.Printf("[k8s] Cannot determine this service's image (%v), build jobs use %s; set BUILDER_IMAGE to override\n", err, defaultBuilderImage)
		return defaultBuilderImage
	}
	c.builderImage = image
	return image
}

// ownImage reads the image of the pod this service runs in (its name is the hostname).
func (c *RealClient) ownImage(ctx context.Context) (string, error) {
	if !c.inCluster {
		return "", fmt.Errorf("not running in a pod")
	}
	pod, err := os.Hostname()
	if err != nil {
		return "", err
	}
	resp, err := c.doReq(ctx, http.MethodGet, fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", c.namespace, url.PathEscape(pod)), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get pod %s: status %d", pod, resp.StatusCode)
	}
	var p struct {
		Spec struct {
			Containers []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return "", err
	}
	if len(p.Spec.Containers) == 0 || p.Spec.Containers[0].Image == "" {
		return "", fmt.Errorf("pod %s has no container image", pod)
	}
	return p.Spec.Containers[0].Image, nil
}

// newDockerRunner connects to Docker; build containers run this very binary in builder mode.
func newDockerRunner(cfg *config.Config) (*DockerClient, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate own binary: %w", err)
	}
	if err := CheckStaticBinary(exe); err != nil {
		return nil, err
	}
	return NewDockerClient(cfg.DockerHost, cfg.RepoName, cfg.RepoDir, cfg.DockerRepoMount, exe)
}
