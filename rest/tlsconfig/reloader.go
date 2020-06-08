package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path"
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
	label          string
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
	closed         bool
}

// NewKeypairReloader return an instance of the TLS cert loader
func NewKeypairReloader(certPath, keyPath string, checkInterval time.Duration) (*KeypairReloader, error) {
	return NewKeypairReloaderWithLabel("", certPath, keyPath, checkInterval)
}

// NewKeypairReloaderWithLabel return an instance of the TLS cert loader
func NewKeypairReloaderWithLabel(label, certPath, keyPath string, checkInterval time.Duration) (*KeypairReloader, error) {
	if label == "" {
		label = path.Base(certPath)
	}

	result := &KeypairReloader{
		label:    label,
		certPath: certPath,
		keyPath:  keyPath,
		stopChan: make(chan struct{}),
	}

	logger.Infof("api=NewKeypairReloader, label=%s, status=started", label)

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
				logger.Infof("api=NewKeypairReloader, status=closed, label=%s, count=%d", result.label, result.LoadedCount())
				return
			case <-tickChan:
				modified := false
				fi, err := os.Stat(certPath)
				if err == nil {
					modified = fi.ModTime().After(result.certModifiedAt)
				} else {
					logger.Warningf("api=NewKeypairReloader, reason=stat, label=%s, file=%q, err=[%v]", result.label, certPath, err)
				}
				if !modified {
					fi, err = os.Stat(keyPath)
					if err == nil {
						modified = fi.ModTime().After(result.keyModifiedAt)
					} else {
						logger.Warningf("api=NewKeypairReloader, reason=stat, label=%s, file=%q, err=[%v]", result.label, keyPath, err)
					}
				}
				// reload on modified, or force to reload each hour
				if modified || result.loadedAt.Add(1*time.Hour).Before(time.Now().UTC()) {
					err := result.Reload()
					if err != nil {
						logger.Errorf("api=NewKeypairReloader, label=%s, err=[%v]", result.label, errors.ErrorStack(err))
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
		logger.Warningf("api=Reload, reason=LoadX509KeyPair, label=%s, file=%q, err=[%v]", k.label, k.certPath, err)
	}
	if err != nil {
		return errors.Annotatef(err, "count: %d", k.count)
	}

	k.keypair = &newCert
	if newCert.Leaf == nil && len(newCert.Certificate) > 0 {
		newCert.Leaf, err = x509.ParseCertificate(newCert.Certificate[0])
		if err != nil {
			logger.Warningf("api=Reload, reason=ParseCertificate, label=%s, err=[%v]", k.label, err)
		}
	}
	if newCert.Leaf != nil {
		logger.Noticef("api=Reload, reason=loaded, label=%s, cn=%q, expires=%q",
			k.label, newCert.Leaf.Subject.CommonName, newCert.Leaf.NotAfter.Format(time.RFC3339))
	}

	certFileInfo, err := os.Stat(k.certPath)
	if err == nil {
		k.certModifiedAt = certFileInfo.ModTime()
	} else {
		logger.Warningf("api=Reload, reason=stat, label=%s, file=%q, err=[%v]", k.label, k.certPath, err)
	}

	keyFileInfo, err := os.Stat(k.keyPath)
	if err == nil {
		k.keyModifiedAt = keyFileInfo.ModTime()
	} else {
		logger.Warningf("api=Reload, reason=stat, label=%s, file=%q, err=[%v]", k.label, k.keyPath, err)
	}

	logger.Infof("api=Reload, label=%s, count=%d, cert=%q, key=%q", k.label, k.count, k.certPath, k.keyPath)
	return nil
}

func (k *KeypairReloader) tlsCert() (*tls.Certificate, error) {
	var err error
	kp := k.Keypair()
	if kp.Leaf == nil && len(kp.Certificate) > 0 {
		kp.Leaf, err = x509.ParseCertificate(kp.Certificate[0])
		if err != nil {
			logger.Warningf("api=tlsCert, reason=ParseCertificate, label=%s, err=[%v]", k.label, err)
		}
	}

	if kp.Leaf != nil {
		if kp.Leaf.NotAfter.Add(1 * time.Hour).Before(time.Now().UTC()) {
			logger.Warningf("api=tlsCert, label=%s, count=%d, cert=%q, expires=%q",
				k.label, k.count, k.certPath, kp.Leaf.NotAfter.Format(time.RFC3339))
		} else {
			logger.Tracef("api=tlsCert, label=%s, count=%d, cert=%q, expires=%q",
				k.label, k.count, k.certPath, kp.Leaf.NotAfter.Format(time.RFC3339))
		}
	}
	return kp, nil
}

// GetKeypairFunc is a callback for TLSConfig to provide TLS certificate and key pair for Server
func (k *KeypairReloader) GetKeypairFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return k.tlsCert()
	}
}

// GetClientCertificateFunc is a callback for TLSConfig to provide TLS certificate and key pair for Client
func (k *KeypairReloader) GetClientCertificateFunc() func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		return k.tlsCert()
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
func (k *KeypairReloader) Close() error {
	if k == nil {
		return nil
	}

	k.lock.RLock()
	defer k.lock.RUnlock()

	if k.closed {
		return errors.New("already closed")
	}

	logger.Infof("api=Close, label=%s, count=%d, cert=%q, key=%q", k.label, k.count, k.certPath, k.keyPath)

	k.closed = true
	k.stopChan <- struct{}{}

	return nil
}
