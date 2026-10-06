package config

import (
	"flag"
	"os"
	"path/filepath"
	"sync"
)

// Config represents runtime configuration.
type Config struct {
	mu sync.RWMutex

	ListenAddr        string
	RepoDir           string
	RepoName          string
	Namespace         string
	CronJobName       string
	Kubeconfig        string
	BuildNode         string
	AutoCheckInterval string
	DevMode           bool
}

// SettingsDTO models dynamically viewable and configurable settings.
type SettingsDTO struct {
	AutoCheckInterval string `json:"autoCheckInterval"`
	BuildNode         string `json:"buildNode"`
	RepoName          string `json:"repoName"`
	Namespace         string `json:"namespace"`
	ListenAddr        string `json:"listenAddr"`
	RepoDir           string `json:"repoDir"`
}

// GetSettings returns a snapshot of runtime settings.
func (c *Config) GetSettings() SettingsDTO {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return SettingsDTO{
		AutoCheckInterval: c.AutoCheckInterval,
		BuildNode:         c.BuildNode,
		RepoName:          c.RepoName,
		Namespace:         c.Namespace,
		ListenAddr:        c.ListenAddr,
		RepoDir:           c.RepoDir,
	}
}

// SetBuildNode updates the build node dynamically.
func (c *Config) SetBuildNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.BuildNode = node
}

// SetAutoCheckInterval updates the auto-check interval string.
func (c *Config) SetAutoCheckInterval(interval string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AutoCheckInterval = interval
}

// Load loads configuration from environment variables and command line flags.
func Load() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultKubeconfig := ""
	if homeDir != "" {
		defaultKubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	cfg := &Config{}

	flag.StringVar(&cfg.ListenAddr, "listen", getEnv("LISTEN_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.RepoDir, "repo-dir", getEnv("REPO_DIR", "/repo"), "Filesystem path to package repository")
	flag.StringVar(&cfg.RepoName, "repo-name", getEnv("REPO_NAME", "zfslocal"), "Name of pacman repository")
	flag.StringVar(&cfg.Namespace, "namespace", getEnv("K8S_NAMESPACE", "arch-repo"), "Kubernetes namespace for build jobs")
	flag.StringVar(&cfg.CronJobName, "cronjob", getEnv("CRONJOB_NAME", "zfs-repo-builder"), "Kubernetes CronJob name to clone jobs from")
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", getEnv("KUBECONFIG", defaultKubeconfig), "Path to kubeconfig file (for out-of-cluster dev)")
	flag.StringVar(&cfg.BuildNode, "build-node", getEnv("BUILD_NODE", ""), "Host to pin build jobs to via nodeSelector (optional)")
	flag.StringVar(&cfg.AutoCheckInterval, "auto-check-interval", getEnv("AUTO_CHECK_INTERVAL", "6h"), "Interval to check Arch upstream for new kernel versions (e.g. 6h, 12h, 24h, or 0 to disable)")
	flag.BoolVar(&cfg.DevMode, "dev", getEnvBool("DEV_MODE", false), "Run in development mode (mock K8s when cluster unavailable)")

	flag.Parse()

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "true" || val == "1" || val == "yes" {
		return true
	}
	if val == "false" || val == "0" || val == "no" {
		return false
	}
	return defaultVal
}
