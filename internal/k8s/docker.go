package k8s

import (
	"archive/tar"
	"bytes"
	"context"
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/kernel"
)

const (
	dockerImage      = "archlinux"
	dockerImageTag   = "base-devel"
	dockerSocketPath = "/var/run/docker.sock"
	dockerLabelName  = "app.kubernetes.io/name"
	dockerLabelValue = "zfs-repo-builder"
)

// DockerClient implements K8sClient by running each build as a container on the
// local Docker Engine (via its HTTP API), for hosts without Kubernetes.
type DockerClient struct {
	httpClient *http.Client
	baseURL    string
	repoName   string
	repoDir    string
	// repoMount is mounted at /repo inside build containers. It must be valid from the
	// Docker host's point of view (host path or named volume).
	repoMount string
	// builderBinary is copied into every build container (see uploadBuilder).
	builderBinary string
}

// NewDockerClient connects to the Docker Engine at host ("" uses DOCKER_HOST or the default
// socket) and verifies it answers. repoMount defaults to repoDir. builderBinary is the statically
// linked binary of this service that build containers run in builder mode.
func NewDockerClient(host, repoName, repoDir, repoMount, builderBinary string) (*DockerClient, error) {
	if host == "" {
		host = os.Getenv("DOCKER_HOST")
	}
	if host == "" {
		host = "unix://" + dockerSocketPath
	}
	if repoMount == "" {
		repoMount = repoDir
	}

	c := &DockerClient{repoName: repoName, repoDir: repoDir, repoMount: repoMount, builderBinary: builderBinary}
	switch {
	case strings.HasPrefix(host, "unix://") || strings.HasPrefix(host, "/"):
		socket := strings.TrimPrefix(host, "unix://")
		c.baseURL = "http://docker"
		c.httpClient = &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		}}
	case strings.HasPrefix(host, "tcp://"):
		c.baseURL = "http://" + strings.TrimPrefix(host, "tcp://")
		c.httpClient = &http.Client{}
	default:
		return nil, fmt.Errorf("unsupported docker host %q (use unix:// or tcp://)", host)
	}

	if err := c.ping(); err != nil {
		return nil, fmt.Errorf("docker engine not reachable at %s: %w", host, err)
	}
	return c, nil
}

func (c *DockerClient) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := c.do(ctx, http.MethodGet, "/_ping", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *DockerClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

// apiError turns an unexpected Docker response into an error.
func apiError(what string, resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("%s (%d): %s", what, resp.StatusCode, strings.TrimSpace(string(b)))
}

// SetBuildNode is a no-op: builds always run on the local Docker host.
func (c *DockerClient) SetBuildNode(string) {}

type dockerInspect struct {
	ID      string `json:"Id"`
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State   struct {
		Status     string `json:"Status"`
		ExitCode   int    `json:"ExitCode"`
		StartedAt  string `json:"StartedAt"`
		FinishedAt string `json:"FinishedAt"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

func parseDockerTime(s string) *time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.IsZero() || t.Year() <= 1 {
		return nil
	}
	return &t
}

func (i *dockerInspect) summary() JobSummary {
	name := strings.TrimPrefix(i.Name, "/")
	status := StatusFailed
	switch i.State.Status {
	case "created":
		status = StatusPending
	case "running", "restarting", "paused":
		status = StatusRunning
	case "exited":
		if i.State.ExitCode == 0 {
			status = StatusSucceeded
		}
	}

	start := parseDockerTime(i.State.StartedAt)
	if start == nil {
		start = parseDockerTime(i.Created)
	}
	var end *time.Time
	if status == StatusSucceeded || status == StatusFailed {
		end = parseDockerTime(i.State.FinishedAt)
	}
	var duration int64
	if start != nil {
		if end != nil {
			duration = int64(end.Sub(*start).Seconds())
		} else if status == StatusRunning {
			duration = int64(time.Since(*start).Seconds())
		}
	}

	return JobSummary{
		Name:              name,
		Namespace:         "docker",
		Status:            status,
		StartTime:         start,
		EndTime:           end,
		DurationSec:       duration,
		Variant:           i.Config.Labels[annotationVariant],
		KernelVersion:     i.Config.Labels[annotationKernel],
		ZfsVersion:        i.Config.Labels[annotationZfs],
		Task:              i.Config.Labels[annotationTask],
		Package:           i.Config.Labels[annotationPackage],
		UnsupportedKernel: i.Config.Labels[annotationUnsupported] == "true",
	}
}

// inspect returns the build container with the given name. Containers that were
// not created by this service are reported as ErrJobNotFound, so that arbitrary
// containers of the host can never be streamed or deleted through the API.
func (c *DockerClient) inspect(ctx context.Context, name string) (*dockerInspect, error) {
	if !ValidJobName(name) {
		return nil, ErrJobNotFound
	}
	resp, err := c.do(ctx, http.MethodGet, "/containers/"+url.PathEscape(name)+"/json", nil)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrJobNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("inspect container", resp)
	}
	var info dockerInspect
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.Config.Labels[dockerLabelName] != dockerLabelValue {
		return nil, ErrJobNotFound
	}
	return &info, nil
}

func (c *DockerClient) ListJobs() ([]JobSummary, error) {
	ctx := context.Background()
	filters, _ := json.Marshal(map[string][]string{"label": {dockerLabelName + "=" + dockerLabelValue}})
	resp, err := c.do(ctx, http.MethodGet, "/containers/json?all=true&filters="+url.QueryEscape(string(filters)), nil)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("list containers", resp)
	}
	var items []struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	summaries := make([]JobSummary, 0, len(items))
	for _, it := range items {
		r, err := c.do(ctx, http.MethodGet, "/containers/"+it.ID+"/json", nil)
		if err != nil {
			return nil, fmt.Errorf("inspect container: %w", err)
		}
		var info dockerInspect
		err = json.NewDecoder(r.Body).Decode(&info)
		r.Body.Close()
		if err != nil || r.StatusCode != http.StatusOK {
			continue // removed in the meantime
		}
		summaries = append(summaries, info.summary())
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].StartTime == nil || summaries[j].StartTime == nil {
			return false
		}
		return summaries[i].StartTime.After(*summaries[j].StartTime)
	})
	return summaries, nil
}

func (c *DockerClient) pullImage(ctx context.Context) error {
	q := url.Values{"fromImage": {dockerImage}, "tag": {dockerImageTag}}
	resp, err := c.do(ctx, http.MethodPost, "/images/create?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return apiError("pull image", resp)
	}
	// The pull is finished when the progress stream ends; errors are reported inside it.
	dec := json.NewDecoder(resp.Body)
	for {
		var msg struct {
			Error string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("pull image: %w", err)
		}
		if msg.Error != "" {
			return fmt.Errorf("pull image: %s", msg.Error)
		}
	}
}

func (c *DockerClient) createContainer(ctx context.Context, name string, body map[string]any) error {
	resp, err := c.do(ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(name), body)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusNotFound:
		return errImageMissing
	default:
		return apiError("create container", resp)
	}
}

var errImageMissing = fmt.Errorf("image not found")

// TriggerBuild starts a build container running the same scripts as the Kubernetes Job.
func (c *DockerClient) TriggerBuild(req BuildRequest) (*JobSummary, error) {
	ctx := context.Background()
	jobName := newJobName()
	req.KernelVersion = kernel.Normalize(req.KernelVersion)

	var env []string
	for _, e := range buildEnv(jobName, c.repoName, req) {
		env = append(env, e["name"]+"="+e["value"])
	}

	body := map[string]any{
		"Image": dockerImage + ":" + dockerImageTag,
		"Cmd":   []string{"/bin/bash", "-c", installScriptsCommand() + "exec " + strings.Join(jobCommand(req), " ")},
		"Env":   env,
		// A TTY makes the log stream plain text instead of Docker's multiplexed frames.
		"Tty": true,
		"Labels": map[string]string{
			dockerLabelName:       dockerLabelValue,
			annotationVariant:     req.Variant,
			annotationKernel:      req.KernelVersion,
			annotationZfs:         req.ZfsVersion,
			annotationTask:        req.Task,
			annotationPackage:     req.Package,
			annotationUnsupported: strconv.FormatBool(req.AllowUnsupportedKernel),
		},
		"HostConfig": map[string]any{
			"Binds": []string{c.repoMount + ":/repo"},
		},
	}

	err := c.createContainer(ctx, jobName, body)
	if err == errImageMissing {
		if err = c.pullImage(ctx); err == nil {
			err = c.createContainer(ctx, jobName, body)
		}
	}
	if err != nil {
		return nil, err
	}
	if err := c.uploadBuilder(ctx, jobName); err != nil {
		c.DeleteJob(jobName)
		return nil, err
	}

	resp, err := c.do(ctx, http.MethodPost, "/containers/"+jobName+"/start", nil)
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		return nil, apiError("start container", resp)
	}

	now := time.Now()
	return &JobSummary{
		Name:              jobName,
		Namespace:         "docker",
		Status:            StatusRunning,
		StartTime:         &now,
		Variant:           req.Variant,
		KernelVersion:     req.KernelVersion,
		ZfsVersion:        req.ZfsVersion,
		Task:              req.Task,
		Package:           req.Package,
		UnsupportedKernel: req.AllowUnsupportedKernel,
	}, nil
}

// DeleteJob removes a (finished or running) build container.
func (c *DockerClient) DeleteJob(name string) error {
	ctx := context.Background()
	if _, err := c.inspect(ctx, name); err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodDelete, "/containers/"+url.PathEscape(name)+"?force=true", nil)
	if err != nil {
		return fmt.Errorf("delete container: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return ErrJobNotFound
	default:
		return apiError("delete container", resp)
	}
}

// StreamLogs follows the container output, or replays the persisted log file of a finished build.
func (c *DockerClient) StreamLogs(ctx context.Context, jobName string) (<-chan string, error) {
	info, err := c.inspect(ctx, jobName)
	if err != nil {
		return nil, err
	}

	logFilePath := filepath.Join(c.repoDir, "logs", jobName+".log")
	if s := info.State.Status; s != "running" && s != "created" && s != "restarting" {
		if lines, fileErr := streamFromFile(ctx, logFilePath); fileErr == nil {
			return lines, nil
		}
	}

	resp, err := c.do(ctx, http.MethodGet, "/containers/"+url.PathEscape(jobName)+"/logs?follow=true&stdout=true&stderr=true", nil)
	if err != nil {
		return nil, fmt.Errorf("stream logs: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, apiError("stream logs", resp)
	}

	return streamLines(ctx, resp.Body), nil
}

// uploadBuilder copies this service's binary into the created container at builderPath, where the job
// runs it in builder mode. The binary must be statically linked to run in the Arch Linux image.
func (c *DockerClient) uploadBuilder(ctx context.Context, container string) error {
	f, err := os.Open(c.builderBinary)
	if err != nil {
		return fmt.Errorf("open builder binary: %w", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}

	// Docker extracts a tar archive into the container; stream it instead of buffering the binary.
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		dir := strings.TrimPrefix(path.Dir(builderPath), "/")
		err := tw.WriteHeader(&tar.Header{Name: dir + "/", Typeflag: tar.TypeDir, Mode: 0o755, ModTime: st.ModTime()})
		if err == nil {
			err = tw.WriteHeader(&tar.Header{Name: strings.TrimPrefix(builderPath, "/"), Mode: 0o755, Size: st.Size(), ModTime: st.ModTime()})
		}
		if err == nil {
			_, err = io.Copy(tw, f)
		}
		if err == nil {
			err = tw.Close()
		}
		pw.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/containers/"+url.PathEscape(container)+"/archive?path=/", pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-tar")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload builder: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return apiError("upload builder", resp)
	}
	return nil
}

// CheckStaticBinary returns an error if the executable at path is dynamically linked. The builder runs
// this service's binary inside the Arch Linux build image, where the host's libraries are not available.
func CheckStaticBinary(path string) error {
	f, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			return fmt.Errorf("%s is dynamically linked; build it with CGO_ENABLED=0 to run builds in Docker", path)
		}
	}
	return nil
}
