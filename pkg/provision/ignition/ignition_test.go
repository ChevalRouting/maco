package ignition

import (
	"encoding/json"
	"strings"
	"testing"
)

func buildToString(t *testing.T, cfg Config) string {
	t.Helper()
	source := []byte(cfg.Raw)
	if cfg.Raw == "" {
		document, err := cfg.easyButane()
		if err != nil {
			t.Fatalf("build butane: %v", err)
		}

		source = document
	}

	data, err := transpile(source)
	if err != nil {
		t.Fatalf("transpile ignition: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("ignition is not valid JSON: %v", err)
	}

	return string(data)
}

func TestFallbackButaneEnablesPodmanSysextAndDisablesDocker(t *testing.T) {
	out := buildToString(t, Config{Hostname: "host"})

	for _, expected := range []string{
		"podman.socket",
		"/etc/flatcar/enabled-sysext.conf",
		"/etc/containers/policy.json",
		"/etc/extensions/docker-flatcar.raw",
		"/etc/extensions/containerd-flatcar.raw",
		"/dev/null",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("fallback butane missing %q in %s", expected, out)
		}
	}
}

func TestRawButaneTranspiles(t *testing.T) {
	raw := "variant: flatcar\nversion: 1.1.0\nstorage:\n  files:\n    - path: /etc/hello\n      contents:\n        inline: hi\n"
	out := buildToString(t, Config{Raw: raw})
	if !strings.Contains(out, "/etc/hello") {
		t.Fatalf("raw butane not transpiled: %s", out)
	}
}

func TestInvalidButaneErrors(t *testing.T) {
	if _, err := transpile([]byte("variant: flatcar\nversion: 9.9.9\n")); err == nil {
		t.Fatal("expected an error for an unknown butane version")
	}
}

func TestFallbackButaneWritesHostnameAndNetwork(t *testing.T) {
	out := buildToString(t, Config{
		Hostname:    "web-01",
		MAC:         "02:00:00:00:00:01",
		Addresses:   []string{"192.0.2.10/24"},
		Gateway:     "192.0.2.1",
		Nameservers: []string{"1.1.1.1"},
	})
	for _, expected := range []string{"/etc/hostname", "/etc/systemd/network/00-maco.network"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("expected %q in %s", expected, out)
		}
	}
}

func TestFallbackButaneWithoutAddressesHasNoNetworkUnit(t *testing.T) {
	out := buildToString(t, Config{MAC: "02:00:00:00:00:01", Gateway: "192.0.2.1"})
	if strings.Contains(out, "00-maco.network") {
		t.Fatalf("network unit emitted without static addresses: %s", out)
	}
}
