package config

import (
	"os"
	"testing"
)

func TestLoadSettingsACME(t *testing.T) {
	paths := &Paths{Root: t.TempDir()}
	config := `tls:
  acme:
    directory: https://ca.internal/acme/acme/directory
    email: ops@example.com
    domains: [maco.example]
    http_addr: ":8080"
    ca: /etc/ssl/internal.pem
`
	if err := os.WriteFile(paths.ConfigPath(), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := LoadSettings(paths)
	if err != nil {
		t.Fatal(err)
	}

	acmeSettings := settings.TLS.ACME
	if acmeSettings == nil || acmeSettings.Directory != "https://ca.internal/acme/acme/directory" || acmeSettings.CA != "/etc/ssl/internal.pem" || acmeSettings.HTTPAddr != ":8080" {
		t.Fatalf("acme = %+v", acmeSettings)
	}

	if err := SaveTLSSettings(paths, TLSSettings{}); err != nil {
		t.Fatal(err)
	}

	saved, err := LoadSettings(paths)
	if err != nil || saved.TLS.ACME != nil {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
}
