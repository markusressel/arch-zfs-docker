package k8s

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/config"
)

func TestMockClient(t *testing.T) {
	client := NewMockClient()

	// List jobs
	jobs, err := client.ListJobs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) == 0 {
		t.Errorf("expected at least 1 mock job, got %d", len(jobs))
	}

	// Trigger build
	newJob, err := client.TriggerBuild(BuildRequest{
		KernelVersion: "7.2.8.arch1-2",
		Variant:       "lts",
		ForceBuild:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error triggering build: %v", err)
	}
	if newJob.Status != "Running" {
		t.Errorf("expected Running status, got %s", newJob.Status)
	}

	// Stream logs
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logChan, err := client.StreamLogs(ctx, newJob.Name)
	if err != nil {
		t.Fatalf("stream logs failed: %v", err)
	}

	var lines []string
	for line := range logChan {
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		t.Error("expected mock logs to be streamed")
	}

	// SetBuildNode (no-op)
	client.SetBuildNode("worker-1")
}

func TestStreamFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	content := "Line 1\nLine 2\nLine 3\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ch, err := streamFromFile(ctx, logFile)
	if err != nil {
		t.Fatalf("streamFromFile failed: %v", err)
	}

	var readLines []string
	for line := range ch {
		readLines = append(readLines, line)
	}

	if len(readLines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(readLines))
	}
	if readLines[0] != "Line 1" || readLines[1] != "Line 2" || readLines[2] != "Line 3" {
		t.Errorf("unexpected lines: %v", readLines)
	}
}

func TestStreamFromFileNonExistent(t *testing.T) {
	_, err := streamFromFile(context.Background(), "/non/existent/file.log")
	if err == nil {
		t.Error("expected error for non existent file")
	}
}

func TestRealClientSetBuildNode(t *testing.T) {
	client := &RealClient{
		namespace: "arch-repo",
		buildNode: "node-1",
	}

	if client.getBuildNode() != "node-1" {
		t.Errorf("expected node-1, got %s", client.getBuildNode())
	}

	client.SetBuildNode("node-2")
	if client.getBuildNode() != "node-2" {
		t.Errorf("expected node-2, got %s", client.getBuildNode())
	}
}

func TestNewClientDevMode(t *testing.T) {
	cfg := &config.Config{
		DevMode: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := client.(*MockClient); !ok {
		t.Errorf("expected MockClient when DevMode is true")
	}
}

func TestParseKubeconfig(t *testing.T) {
	tmpDir := t.TempDir()
	kcPath := filepath.Join(tmpDir, "config")
	yamlContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
    certificate-authority-data: dGVzdC1jYQ==
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
users:
- name: default
  user:
    token: test-token
`
	if err := os.WriteFile(kcPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	kc, err := parseKubeconfig(kcPath)
	if err != nil {
		t.Fatalf("parseKubeconfig failed: %v", err)
	}

	if kc.server != "https://127.0.0.1:6443" {
		t.Errorf("expected server https://127.0.0.1:6443, got %s", kc.server)
	}
	if kc.token != "test-token" {
		t.Errorf("expected token test-token, got %s", kc.token)
	}
}

func TestRealClientStreamLogsFileFallback(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}

	jobName := "zfs-build-test-123"
	logFile := filepath.Join(logsDir, jobName+".log")
	if err := os.WriteFile(logFile, []byte("Test log line 1\nTest log line 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &RealClient{
		namespace:  "arch-repo",
		repoDir:    tmpDir,
		baseURL:    "http://127.0.0.1:9999",
		httpClient: &http.Client{Timeout: 50 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.StreamLogs(ctx, jobName)
	if err != nil {
		t.Fatalf("StreamLogs failed: %v", err)
	}

	var lines []string
	for l := range ch {
		lines = append(lines, l)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from file fallback, got %d", len(lines))
	}
}

func TestNewJobNameIsUnique(t *testing.T) {
	a, b := newJobName(), newJobName()
	if a == b {
		t.Fatalf("job names collided: %s", a)
	}
	if !validJobName(a) {
		t.Fatalf("generated name %q is not a valid job name", a)
	}
}
