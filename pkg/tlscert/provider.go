package tlscert

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"

	"github.com/ChevalRouting/certs"
	"github.com/m-vinc/maco/pkg/config"
)

var (
	ErrLocked  = errors.New("tls is set by --tls-cert and --tls-key on the command line")
	ErrInvalid = errors.New("invalid tls settings")
)

type Provider struct {
	paths    *config.Paths
	certs    *certs.Provider
	locked   bool
	mu       sync.Mutex
	settings config.TLSSettings
}

func New(paths *config.Paths, settings config.TLSSettings, httpsAddr string) (*Provider, error) {
	provider, err := certs.New(options(paths, httpsAddr), librarySettings(settings))
	if err != nil {
		return nil, err
	}

	return &Provider{paths: paths, certs: provider, settings: settings}, nil
}

func NewLocked(paths *config.Paths, certPath, keyPath, httpsAddr string) (*Provider, error) {
	provider, err := New(paths, config.TLSSettings{Cert: certPath, Key: keyPath}, httpsAddr)
	if err != nil {
		return nil, err
	}

	provider.locked = true
	return provider, nil
}

func options(paths *config.Paths, httpsAddr string) certs.Options {
	return certs.Options{Dir: paths.Root, FallbackCert: paths.TLSCertPath(), FallbackKey: paths.TLSKeyPath(), HTTPSAddr: httpsAddr}
}

func librarySettings(settings config.TLSSettings) certs.Settings {
	converted := certs.Settings{CertFile: settings.Cert, KeyFile: settings.Key}
	if a := settings.ACME; a != nil {
		converted.ACME = &certs.ACME{Directory: a.Directory, Email: a.Email, Domains: a.Domains, ChallengeAddr: a.HTTPAddr, CAFile: a.CA, EABKID: a.EABKID, EABKey: a.EABKey}
	}

	return converted
}

func (p *Provider) Mode() string { return p.certs.Settings().Mode() }

func (p *Provider) TLSConfig() *tls.Config { return p.certs.TLSConfig() }

func (p *Provider) Start(ctx context.Context) { p.certs.Start(ctx) }

func Inspect(paths *config.Paths, settings config.TLSSettings) ([]Status, error) {
	provider, err := certs.New(options(paths, ""), librarySettings(settings))
	if err != nil {
		return nil, err
	}

	return statuses(provider.Status().Certificates), nil
}

func statuses(certificates []certs.Certificate) []Status {
	converted := make([]Status, 0, len(certificates))
	for _, c := range certificates {
		converted = append(converted, Status{Name: c.Name, Source: c.Source, Issuer: c.Issuer, NotAfter: c.NotAfter, Ready: c.Ready, Error: c.Error})
	}

	return converted
}
