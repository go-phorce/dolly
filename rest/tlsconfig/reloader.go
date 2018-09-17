package tlsconfig

import (
	"crypto/tls"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-phorce/pkg/algorithms/throttle"
	"github.com/juju/errors"
)

// KeypairReloader keeps necessary info to provide reloaded certificate
type KeypairReloader struct {
	watcher    *fsnotify.Watcher
	lock       sync.RWMutex
	loadedAt   time.Time
	count      uint32
	keypair    *tls.Certificate
	certPath   string
	keyPath    string
	throttle   throttle.Driver
	inProgress bool
}

const evtToWatch = fsnotify.Write

// NewKeypairReloader return an instance of the TLS cert loader
func NewKeypairReloader(certPath, keyPath string, throttlePeriod time.Duration) (*KeypairReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Trace(err)
	}

	result := &KeypairReloader{
		watcher:  watcher,
		certPath: certPath,
		keyPath:  keyPath,
	}

	logger.Infof("api=NewKeypairReloader, status=started")

	err = result.Reload()
	if err != nil {
		return nil, errors.Trace(err)
	}

	result.throttle = throttle.Func(throttlePeriod, true, func() {
		if err := result.Reload(); err != nil {
			logger.Errorf("api=NewKeypairReloader, reason=reload, cert='%s', key='%s', err=[%v]", certPath, keyPath, errors.ErrorStack(err))
		}
	})

	go func() {
		closed := false
		for !closed {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					closed = true
				}
				if event.Op&evtToWatch != 0 {
					logger.Noticef("api=NewKeypairReloader, reason=event, event=[%v]", event)
					result.throttle.Trigger()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					closed = true
				}
				if err != nil {
					logger.Errorf("api=NewKeypairReloader, reason=watcher,  cert='%s', key='%s', err=[%v]", certPath, keyPath, errors.ErrorStack(err))
				}
			}
		}
		logger.Infof("api=NewKeypairReloader, status=closed, count=%d", result.LoadedCount())
	}()

	watcher.Add(certPath)
	watcher.Add(keyPath)

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
		logger.Errorf("api=NewKeypairReloader, reason=LoadX509KeyPair, err=[%v]", err)
	}
	if err != nil {
		return errors.Annotatef(err, "count: %d", k.count)
	}

	k.keypair = &newCert

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

	if k.throttle != nil {
		k.throttle.Stop()
	}

	if k.watcher != nil {
		k.watcher.Close()
		k.watcher = nil
	}
}
