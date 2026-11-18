package tlscert

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/certs"
	"github.com/m-vinc/maco/pkg/config"
)

func pemPair(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	crt, key := filepath.Join(dir, "pair.crt"), filepath.Join(dir, "pair.key")
	if err := certs.EnsureSelfSigned(crt, key); err != nil {
		t.Fatal(err)
	}

	return readText(crt), readText(key)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	addr := listener.Addr().String()
	_ = listener.Close()
	return addr
}

func TestUpdateSwitchesModesAndPersists(t *testing.T) {
	paths := &config.Paths{Root: t.TempDir()}
	if err := os.WriteFile(paths.ConfigPath(), []byte("base_mac: 52:54:00:aa:bb:00\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	provider, err := New(paths, config.TLSSettings{}, ":8443")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	provider.Start(ctx)

	certPEM, keyPEM := pemPair(t)
	if err := provider.Update(Update{Mode: "file", CertPEM: certPEM, KeyPEM: keyPEM}); err != nil {
		t.Fatal(err)
	}

	served, err := provider.TLSConfig().GetCertificate(&tls.ClientHelloInfo{ServerName: "maco.example"})
	if err != nil {
		t.Fatal(err)
	}

	pair, _ := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if provider.Mode() != "file" || string(served.Certificate[0]) != string(pair.Certificate[0]) {
		t.Fatal("uploaded certificate is not served")
	}

	if err := provider.Update(Update{Mode: "file"}); err != nil {
		t.Fatalf("keep current files: %v", err)
	}

	caPEM, _ := pemPair(t)
	acmeUpdate := &ACMEUpdate{Directory: "https://ca.invalid/directory", Domains: []string{"maco.example", " "}, HTTPAddr: freeAddr(t), CAPEM: caPEM, EABKID: "kid", EABKey: "c2VjcmV0"}
	if err := provider.Update(Update{Mode: "acme", ACME: acmeUpdate}); err != nil {
		t.Fatal(err)
	}

	acmeUpdate.EABKey = ""
	if err := provider.Update(Update{Mode: "acme", ACME: acmeUpdate}); err != nil {
		t.Fatal(err)
	}

	view := provider.View()
	if view.Mode != "acme" || view.ACME.CAPEM != caPEM || !view.ACME.HasEABKey || len(view.ACME.Domains) != 1 {
		t.Fatalf("view = %+v, acme = %+v", view, view.ACME)
	}

	saved, err := config.LoadSettings(paths)
	if err != nil {
		t.Fatal(err)
	}

	if saved.BaseMAC != "52:54:00:aa:bb:00" || saved.TLS.ACME == nil || saved.TLS.ACME.EABKey != "c2VjcmV0" || saved.TLS.ACME.CA != filepath.Join(paths.TLSDir(), "acme-ca.pem") {
		t.Fatalf("saved = %+v", saved)
	}

	if err := provider.Update(Update{Mode: "self-signed"}); err != nil || provider.Mode() != "self-signed" {
		t.Fatalf("back to self-signed: %v", err)
	}
}

func TestUpdateRejectsInvalidInput(t *testing.T) {
	paths := &config.Paths{Root: t.TempDir()}
	provider, err := New(paths, config.TLSSettings{}, ":8443")
	if err != nil {
		t.Fatal(err)
	}

	certPEM, _ := pemPair(t)
	_, otherKey := pemPair(t)
	updates := []Update{
		{Mode: "file", CertPEM: certPEM, KeyPEM: otherKey},
		{Mode: "file"},
		{Mode: "acme"},
		{Mode: "acme", ACME: &ACMEUpdate{Domains: []string{"localhost"}}},
		{Mode: "acme", ACME: &ACMEUpdate{Domains: []string{"maco.example"}, CAPEM: "not a pem"}},
		{Mode: "acme", ACME: &ACMEUpdate{Domains: []string{"maco.example"}, EABKID: "kid", EABKey: "%%%"}},
		{Mode: "other"},
	}

	for _, update := range updates {
		if err := provider.Update(update); !errors.Is(err, ErrInvalid) {
			t.Errorf("update %+v: error = %v, want ErrInvalid", update, err)
		}
	}

	if provider.Mode() != "self-signed" {
		t.Fatalf("mode changed to %q after rejected updates", provider.Mode())
	}

	if _, err := os.Stat(paths.ConfigPath()); !os.IsNotExist(err) {
		t.Fatal("rejected updates wrote config.yaml")
	}
}

func TestLockedProviderRefusesUpdates(t *testing.T) {
	paths := &config.Paths{Root: t.TempDir()}
	crt, key := filepath.Join(paths.Root, "flag.crt"), filepath.Join(paths.Root, "flag.key")
	if err := certs.EnsureSelfSigned(crt, key); err != nil {
		t.Fatal(err)
	}

	provider, err := NewLocked(paths, crt, key, ":8443")
	if err != nil {
		t.Fatal(err)
	}

	if err := provider.Update(Update{Mode: "self-signed"}); !errors.Is(err, ErrLocked) || !provider.View().Locked {
		t.Fatalf("locked update error = %v", err)
	}
}

func TestInspect(t *testing.T) {
	paths := &config.Paths{Root: t.TempDir()}
	statuses, err := Inspect(paths, config.TLSSettings{ACME: &config.ACMESettings{Domains: []string{"Maco.Example."}}})
	if err != nil {
		t.Fatal(err)
	}

	if len(statuses) != 2 || statuses[0].Name != "fallback" || !statuses[0].Ready || statuses[1].Name != "maco.example" || statuses[1].Ready {
		t.Fatalf("statuses = %+v", statuses)
	}
}
