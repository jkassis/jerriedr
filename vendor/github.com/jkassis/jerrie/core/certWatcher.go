package core

import (
	"crypto/tls"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// A CertWatcher represents a certificate manager able to watch certificate
// and key pairs for changes.
type CertWatcher struct {
	mu       sync.RWMutex
	certFile string
	keyFile  string
	keyPair  *tls.Certificate
	watcher  *fsnotify.Watcher
	watching chan bool
	log      *logrus.Logger
}

// New creates a new CertWatcher. The certFile and the keyFile
// are both paths to the location of the files. Relative and
// absolute paths are accepted.
func CertWatcherNew(certFile, keyFile string, log *logrus.Logger) (*CertWatcher, error) {
	var err error
	certFile, err = filepath.Abs(certFile)
	if err != nil {
		return nil, err
	}
	keyFile, err = filepath.Abs(keyFile)
	if err != nil {
		return nil, err
	}
	cm := &CertWatcher{
		mu:       sync.RWMutex{},
		certFile: certFile,
		keyFile:  keyFile,
		log:      log,
	}
	return cm, nil
}

// Watch starts watching for changes to the certificate
// and key files. On any change the certificate and key
// are reloaded. If there is an issue the load will fail
// and the old (if any) certificates and keys will continue
// to be used.
func (cm *CertWatcher) Watch() error {
	var err error
	if cm.watcher, err = fsnotify.NewWatcher(); err != nil {
		return errors.Wrap(err, "CertWatcher: can't create watcher")
	}
	if err = cm.watcher.Add(cm.certFile); err != nil {
		return errors.Wrap(err, "CertWatcher: can't watch cert file")
	}
	if err = cm.watcher.Add(cm.keyFile); err != nil {
		return errors.Wrap(err, "CertWatcher: can't watch key file")
	}
	if err := cm.load(); err != nil {
		cm.log.Printf("CertWatcher: can't load cert or key file: %v", err)
	}
	cm.log.Printf("CertWatcher: watching for cert and key change")
	cm.watching = make(chan bool)
	go cm.run()
	return nil
}

func (cm *CertWatcher) load() error {
	keyPair, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
	if err == nil {
		cm.mu.Lock()
		cm.keyPair = &keyPair
		cm.mu.Unlock()
		cm.log.Printf("CertWatcher: certificate and key loaded")
	}
	return err
}

func (cm *CertWatcher) run() {
loop:
	for {
		select {
		case <-cm.watching:
			break loop
		case event := <-cm.watcher.Events:
			cm.log.Printf("CertWatcher: watch event: %v", event)
			if err := cm.load(); err != nil {
				cm.log.Printf("CertWatcher: can't load cert or key file: %v", err)
			}
		case err := <-cm.watcher.Errors:
			cm.log.Printf("CertWatcher: error watching files: %v", err)
		}
	}
	cm.log.Printf("CertWatcher: stopped watching")
	cm.watcher.Close()
}

// GetCertificate returns the loaded certificate for use by
// the TLSConfig fields GetCertificate field in a http.Server.
func (cm *CertWatcher) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.keyPair, nil
}

// Stop tells CertWatcher to stop watching for changes to the
// certificate and key files.
func (cm *CertWatcher) Stop() {
	cm.watching <- false
}
