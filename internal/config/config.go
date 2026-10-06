package config

import (
	"flag"
	"os"
	"path/filepath"
)

// Config represents runtime configuration.
type Config struct {
	ListenAddr  string
	RepoDir     string
	RepoName    string
	Namespace   string
	CronJobName string
	Kubeconfig  string
	BuildNode         string
	AutoCheckInterval string
	DevMode           bool
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
	flag.StringVar(&cfg.BuildNode, "build-node", getEnv("BUILD_NODE", "kfc"), "Host to pin build jobs to via nodeSelector")
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
