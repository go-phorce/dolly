package cryptoprov

import (
	"sync"

	"github.com/go-phorce/dolly/xpki/crypto11"
	"github.com/pkg/errors"
)

// ProviderLoader is interface for loading provider by manufacturer
type ProviderLoader func(cfg TokenConfig) (Provider, error)

var (
	lockLoaders sync.RWMutex
	loaders     = make(map[string]ProviderLoader)
)

// Register provider loader by manufacturer
func Register(manufacturer string, loader ProviderLoader) error {
	lockLoaders.Lock()
	defer lockLoaders.Unlock()

	if _, ok := loaders[manufacturer]; ok {
		return errors.Errorf("already registered: %s", manufacturer)
	}

	loaders[manufacturer] = loader

	return nil
}

// Unregister provider loader by manufacturer
func Unregister(manufacturer string) (ProviderLoader, error) {
	lockLoaders.Lock()
	defer lockLoaders.Unlock()

	if loader, ok := loaders[manufacturer]; ok {
		delete(loaders, manufacturer)
		return loader, nil
	}

	return nil, errors.Errorf("not registered: %s", manufacturer)
}

// LoadProvider load a single provider
func LoadProvider(configLocation string) (Provider, error) {
	tc, err := LoadTokenConfig(configLocation)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	manufacturer := tc.Manufacturer()
	loader, ok := loaders[manufacturer]
	if !ok {
		return nil, errors.Errorf("provider not registered: %s", manufacturer)
	}

	prov, err := loader(tc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return prov, nil
}

// Load returns Crypto with loaded providers from the given config locations
func Load(defaultConfig string, providersConfigs []string) (*Crypto, error) {
	p, err := LoadProvider(defaultConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	c, err := New(p, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = c.Add(p)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, configLocation := range providersConfigs {
		p, err := LoadProvider(configLocation)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = c.Add(p)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return c, nil
}

// Crypto11Loader provides loader for crypto11 provider
func Crypto11Loader(cfg TokenConfig) (Provider, error) {
	p, err := crypto11.Init(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p, nil
}
