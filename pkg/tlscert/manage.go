package tlscert

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/certs"
	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/storage"
)

type View struct {
	Mode         string   `json:"mode"`
	Locked       bool     `json:"locked"`
	Error        string   `json:"error"`
	CertPEM      string   `json:"cert_pem"`
	ACME         ACMEView `json:"acme"`
	Certificates []Status `json:"certificates"`
}

type ACMEView struct {
	Directory string   `json:"directory"`
	Email     string   `json:"email"`
	Domains   []string `json:"domains"`
	HTTPAddr  string   `json:"http_addr"`
	CAPEM     string   `json:"ca_pem"`
	EABKID    string   `json:"eab_kid"`
	HasEABKey bool     `json:"has_eab_key"`
}

type Update struct {
	Mode    string      `json:"mode"`
	CertPEM string      `json:"cert_pem"`
	KeyPEM  string      `json:"key_pem"`
	ACME    *ACMEUpdate `json:"acme"`
}

type ACMEUpdate struct {
	Directory string   `json:"directory"`
	Email     string   `json:"email"`
	Domains   []string `json:"domains"`
	HTTPAddr  string   `json:"http_addr"`
	CAPEM     string   `json:"ca_pem"`
	EABKID    string   `json:"eab_kid"`
	EABKey    string   `json:"eab_key"`
}

type Status struct {
	Name     string    `json:"name"`
	Source   string    `json:"source"`
	Issuer   string    `json:"issuer"`
	NotAfter time.Time `json:"not_after"`
	Ready    bool      `json:"ready"`
	Error    string    `json:"error"`
}

func (p *Provider) View() View {
	p.mu.Lock()
	settings := p.settings
	p.mu.Unlock()
	status := p.certs.Status()
	view := View{Mode: status.Mode, Locked: p.locked, Error: status.Error, Certificates: statuses(status.Certificates)}
	if settings.Cert != "" {
		view.CertPEM = readText(settings.Cert)
	}

	acmeSettings := librarySettings(settings).ACME
	if acmeSettings == nil {
		acmeSettings = &certs.ACME{Domains: []string{}}
	}

	view.ACME = ACMEView{
		Directory: acmeSettings.DirectoryURL(),
		Email:     acmeSettings.Email,
		Domains:   acmeSettings.Domains,
		HTTPAddr:  acmeSettings.ListenAddr(),
		EABKID:    acmeSettings.EABKID,
		HasEABKey: acmeSettings.EABKey != "",
	}

	if acmeSettings.CAFile != "" {
		view.ACME.CAPEM = readText(acmeSettings.CAFile)
	}

	return view
}

func (p *Provider) Update(update Update) error {
	if p.locked {
		return ErrLocked
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	settings, err := p.resolve(update)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalid, err)
	}

	if err := p.certs.Apply(librarySettings(settings)); err != nil {
		return err
	}

	p.settings = settings
	return config.SaveTLSSettings(p.paths, settings)
}

func (p *Provider) resolve(update Update) (config.TLSSettings, error) {
	current := p.settings
	switch update.Mode {
	case "self-signed":
		return config.TLSSettings{}, nil
	case "file":
		return p.resolveFile(update, current)
	case "acme":
		if update.ACME == nil {
			return config.TLSSettings{}, errors.New("acme settings are required")
		}

		return p.resolveACME(update.ACME, current.ACME)
	default:
		return config.TLSSettings{}, fmt.Errorf("unknown mode %q", update.Mode)
	}
}

func (p *Provider) resolveFile(update Update, current config.TLSSettings) (config.TLSSettings, error) {
	if update.CertPEM == "" && update.KeyPEM == "" && current.Cert != "" {
		return config.TLSSettings{Cert: current.Cert, Key: current.Key}, nil
	}

	if _, err := tls.X509KeyPair([]byte(update.CertPEM), []byte(update.KeyPEM)); err != nil {
		return config.TLSSettings{}, fmt.Errorf("certificate and key: %w", err)
	}

	certPath, keyPath := filepath.Join(p.paths.TLSDir(), "custom.crt"), filepath.Join(p.paths.TLSDir(), "custom.key")
	if err := storage.WriteFile(keyPath, []byte(update.KeyPEM), 0o600); err != nil {
		return config.TLSSettings{}, err
	}

	if err := storage.WriteFile(certPath, []byte(update.CertPEM), 0o644); err != nil {
		return config.TLSSettings{}, err
	}

	return config.TLSSettings{Cert: certPath, Key: keyPath}, nil
}

func (p *Provider) resolveACME(update *ACMEUpdate, current *config.ACMESettings) (config.TLSSettings, error) {
	acmeSettings := &config.ACMESettings{
		Directory: strings.TrimSpace(update.Directory),
		Email:     strings.TrimSpace(update.Email),
		Domains:   cleanDomains(update.Domains),
		HTTPAddr:  strings.TrimSpace(update.HTTPAddr),
		EABKID:    strings.TrimSpace(update.EABKID),
		EABKey:    strings.TrimSpace(update.EABKey),
	}

	if acmeSettings.EABKey == "" && current != nil && acmeSettings.EABKID != "" && acmeSettings.EABKID == current.EABKID {
		acmeSettings.EABKey = current.EABKey
	}

	settings := config.TLSSettings{ACME: acmeSettings}
	if err := librarySettings(settings).Validate(); err != nil {
		return config.TLSSettings{}, err
	}

	if strings.TrimSpace(update.CAPEM) == "" {
		return settings, nil
	}

	if !x509.NewCertPool().AppendCertsFromPEM([]byte(update.CAPEM)) {
		return config.TLSSettings{}, errors.New("no certificate found in the acme ca bundle")
	}

	if current != nil && current.CA != "" && readText(current.CA) == update.CAPEM {
		acmeSettings.CA = current.CA
		return settings, nil
	}

	acmeSettings.CA = filepath.Join(p.paths.TLSDir(), "acme-ca.pem")
	if err := storage.WriteFile(acmeSettings.CA, []byte(update.CAPEM), 0o644); err != nil {
		return config.TLSSettings{}, err
	}

	return settings, nil
}

func cleanDomains(domains []string) []string {
	cleaned := make([]string, 0, len(domains))
	for _, domain := range domains {
		if domain = strings.TrimSpace(domain); domain != "" {
			cleaned = append(cleaned, domain)
		}
	}

	return cleaned
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(data)
}
