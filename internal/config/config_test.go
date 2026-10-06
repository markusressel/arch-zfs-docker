package config

import (
	"os"
	"testing"
)

func TestConfigSettings(t *testing.T) {
	cfg := &Config{
		ListenAddr:        ":8080",
		RepoDir:           "/repo",
		RepoName:          "zfslocal",
		Namespace:         "arch-repo",
		BuildNode:         "worker-1",
		AutoCheckInterval: "6h",
	}

	settings := cfg.GetSettings()
	if settings.RepoName != "zfslocal" {
		t.Errorf("expected zfslocal, got %s", settings.RepoName)
	}
	if settings.BuildNode != "worker-1" {
		t.Errorf("expected worker-1, got %s", settings.BuildNode)
	}

	cfg.SetBuildNode("worker-2")
	if cfg.GetSettings().BuildNode != "worker-2" {
		t.Errorf("expected worker-2, got %s", cfg.GetSettings().BuildNode)
	}

	cfg.SetAutoCheckInterval("12h")
	if cfg.GetSettings().AutoCheckInterval != "12h" {
		t.Errorf("expected 12h, got %s", cfg.GetSettings().AutoCheckInterval)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY_EXISTING", "hello")
	defer os.Unsetenv("TEST_KEY_EXISTING")

	if val := getEnv("TEST_KEY_EXISTING", "default"); val != "hello" {
		t.Errorf("expected hello, got %s", val)
	}
	if val := getEnv("TEST_KEY_NONEXISTENT", "default"); val != "default" {
		t.Errorf("expected default, got %s", val)
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE_1", "true")
	os.Setenv("TEST_BOOL_TRUE_2", "1")
	os.Setenv("TEST_BOOL_TRUE_3", "yes")
	os.Setenv("TEST_BOOL_FALSE_1", "false")
	os.Setenv("TEST_BOOL_FALSE_2", "0")
	os.Setenv("TEST_BOOL_FALSE_3", "no")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE_1")
		os.Unsetenv("TEST_BOOL_TRUE_2")
		os.Unsetenv("TEST_BOOL_TRUE_3")
		os.Unsetenv("TEST_BOOL_FALSE_1")
		os.Unsetenv("TEST_BOOL_FALSE_2")
		os.Unsetenv("TEST_BOOL_FALSE_3")
	}()

	if !getEnvBool("TEST_BOOL_TRUE_1", false) {
		t.Error("expected true")
	}
	if !getEnvBool("TEST_BOOL_TRUE_2", false) {
		t.Error("expected true")
	}
	if !getEnvBool("TEST_BOOL_TRUE_3", false) {
		t.Error("expected true")
	}
	if getEnvBool("TEST_BOOL_FALSE_1", true) {
		t.Error("expected false")
	}
	if getEnvBool("TEST_BOOL_FALSE_2", true) {
		t.Error("expected false")
	}
	if getEnvBool("TEST_BOOL_FALSE_3", true) {
		t.Error("expected false")
	}
	if !getEnvBool("TEST_BOOL_NONE", true) {
		t.Error("expected default true")
	}
}
