package tlsconfig

import (
	"crypto/tls"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
)

// Wrap time.Tick so we can override it in tests.
var makeTicker = func(interval time.Duration) (func(), <-chan time.Time) {
	t := time.NewTicker(interval)
	return t.Stop, t.C
}

// KeypairReloader keeps necessary info to provide reloaded certificate
type KeypairReloader struct {
	lock           sync.RWMutex
	loadedAt       time.Time
	count          uint32
	keypair        *tls.Certificate
	certPath       string
	certModifiedAt time.Time
	keyPath        string
	keyModifiedAt  time.Time
	inProgress     bool
	stopChan       chan<- struct{}
}

// NewKeypairReloader return an instance of the TLS cert loader
func NewKeypairReloader(certPath, keyPath string, checkInterval time.Duration) (*KeypairReloader, error) {
	result := &KeypairReloader{
		certPath: certPath,
		keyPath:  keyPath,
		stopChan: make(chan struct{}),
	}

	logger.Infof("api=NewKeypairReloader, status=started")

	err := result.Reload()
	if err != nil {
		return nil, errors.Trace(err)
	}

	stopChan := make(chan struct{})
	tickerStop, tickChan := makeTicker(checkInterval)
	go func() {
		for {
			select {
			case <-stopChan:
				tickerStop()
				logger.Infof("api=NewKeypairReloader, status=closed, count=%d", result.LoadedCount())
				return
			case <-tickChan:
				modified := false
				fi, err := os.Stat(certPath)
				if err == nil {
					modified = fi.ModTime().After(result.certModifiedAt)
				} else {
					logger.Warning("api=NewKeypairReloader, reason=stat, file=%q, err=[%v]", certPath, err)
				}
				if !modified {
					fi, err = os.Stat(keyPath)
					if err == nil {
						modified = fi.ModTime().After(result.keyModifiedAt)
					} else {
						logger.Warning("api=NewKeypairReloader, reason=stat, file=%q, err=[%v]", keyPath, err)
					}
				}
				if modified {
					err := result.Reload()
					if err != nil {
						logger.Errorf("api=NewKeypairReloader, err=[%v]", errors.ErrorStack(err))
					}
				}
			}
		}
	}()
	result.stopChan = stopChan
	return result, nil
}

// Reload will explicitly load TLS certs from the disk
func (k *KeypairReloader) Reload() error {
	k.lock.Lock()
	defer func() {
		k.inProgress = false
		k.lock.Unlock()
	}()

	if k.inProgress {
		return nil
	}

	k.inProgress = true

	atomic.AddUint32(&k.count, 1)
	k.loadedAt = time.Now().UTC()

	var newCert tls.Certificate
	var err error

	for i := 0; i < 3; i++ {
		// sleep a little as notification occurs right after process starts writing the file,
		// so it needs to finish writing the file
		time.Sleep(50 * time.Millisecond)
		newCert, err = tls.LoadX509KeyPair(k.certPath, k.keyPath)
		if err == nil {
			break
		}
		logger.Warning("api=Reload, reason=LoadX509KeyPair, err=[%v]", err)
	}
	if err != nil {
		return errors.Annotatef(err, "count: %d", k.count)
	}

	k.keypair = &newCert

	certFileInfo, err := os.Stat(k.certPath)
	if err == nil {
		k.certModifiedAt = certFileInfo.ModTime()
	} else {
		logger.Warning("api=Reload, reason=stat, file=%q, err=[%v]", k.certPath, err)
	}

	keyFileInfo, err := os.Stat(k.keyPath)
	if err == nil {
		k.keyModifiedAt = keyFileInfo.ModTime()
	} else {
		logger.Warning("api=Reload, reason=stat, file=%q, err=[%v]", k.keyPath, err)
	}

	logger.Infof("api=NewKeypairReloader, count=%d, cert='%s', key='%s'", k.count, k.certPath, k.keyPath)
	return nil
}

// GetKeypairFunc is a callback for TLSConfig to provide TLS certificate and key pair
func (k *KeypairReloader) GetKeypairFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return k.Keypair(), nil
	}
}

// Keypair returns current pair
func (k *KeypairReloader) Keypair() *tls.Certificate {
	if k == nil {
		return nil
	}
	k.lock.RLock()
	defer k.lock.RUnlock()

	return k.keypair
}

// CertAndKeyFiles returns cert and key files
func (k *KeypairReloader) CertAndKeyFiles() (string, string) {
	if k == nil {
		return "", ""
	}
	k.lock.RLock()
	defer k.lock.RUnlock()

	return k.certPath, k.keyPath
}

// LoadedAt return the last time when the pair was loaded
func (k *KeypairReloader) LoadedAt() time.Time {
	k.lock.RLock()
	defer k.lock.RUnlock()

	return k.loadedAt
}

// LoadedCount returns the number of times the pair was loaded from disk
func (k *KeypairReloader) LoadedCount() uint32 {
	return atomic.LoadUint32(&k.count)
}

// Close will close the reloader and release its resources
func (k *KeypairReloader) Close() {
	if k == nil {
		return
	}

	k.lock.RLock()
	defer k.lock.RUnlock()

	k.stopChan <- struct{}{}
}
