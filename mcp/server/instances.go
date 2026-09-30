package mcpserver

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type FileConfig struct {
	Instances map[string]InstanceConfig `yaml:"instances"`
}

type InstanceConfig struct {
	URL      string    `yaml:"url"`
	Token    string    `yaml:"token"`
	TokenEnv string    `yaml:"token_env"`
	TLS      TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	SkipVerify bool `yaml:"skip_verify"`
}

type instance struct {
	name   string
	url    string
	token  string
	client *http.Client
}

func loadInstances(path string) (map[string]*instance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse instance config: %w", err)
	}

	out := make(map[string]*instance, len(cfg.Instances))
	for name, item := range cfg.Instances {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(item.URL) == "" {
			return nil, fmt.Errorf("instance name and url are required")
		}

		if item.Token != "" && item.TokenEnv != "" {
			return nil, fmt.Errorf("instance %q: set token or token_env, not both", name)
		}

		token := item.Token
		if item.TokenEnv != "" {
			token = os.Getenv(item.TokenEnv)
		}

		if token == "" {
			return nil, fmt.Errorf("instance %q: token is required", name)
		}

		parsed, err := url.Parse(item.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("instance %q: url must be an HTTP(S) URL without credentials, query or fragment", name)
		}

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: item.TLS.SkipVerify}
		out[name] = &instance{
			name:  name,
			url:   strings.TrimRight(item.URL, "/"),
			token: token,
			client: &http.Client{
				Timeout:       2 * time.Hour,
				Transport:     transport,
				CheckRedirect: rejectRedirect,
			},
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no instances configured")
	}

	return out, nil
}

func rejectRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
