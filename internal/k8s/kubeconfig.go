package k8s

import (
	"encoding/base64"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type parsedKubeconfig struct {
	server string
	caData []byte
	token  string
}

// Minimal kubeconfig YAML parser for server, ca, and auth token.
type rawKubeconfig struct {
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server                   string `yaml:"server"`
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			Token string `yaml:"token"`
		} `yaml:"user"`
	} `yaml:"users"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
}

func parseKubeconfig(path string) (*parsedKubeconfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw rawKubeconfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	currentCluster := ""
	currentUser := ""
	for _, ctx := range raw.Contexts {
		if ctx.Name == raw.CurrentContext {
			currentCluster = ctx.Context.Cluster
			currentUser = ctx.Context.User
			break
		}
	}

	if currentCluster == "" && len(raw.Clusters) > 0 {
		currentCluster = raw.Clusters[0].Name
	}
	if currentUser == "" && len(raw.Users) > 0 {
		currentUser = raw.Users[0].Name
	}

	var res parsedKubeconfig
	for _, cl := range raw.Clusters {
		if cl.Name == currentCluster {
			res.server = cl.Cluster.Server
			if cl.Cluster.CertificateAuthorityData != "" {
				res.caData, _ = base64.StdEncoding.DecodeString(cl.Cluster.CertificateAuthorityData)
			}
			break
		}
	}

	for _, u := range raw.Users {
		if u.Name == currentUser {
			res.token = u.User.Token
			break
		}
	}

	if res.server == "" {
		return nil, fmt.Errorf("no server found in kubeconfig %s", path)
	}

	return &res, nil
}
