package execution

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var knownHostsMu sync.Mutex

type UnknownHostKeyError struct {
	Hostname          string `json:"hostname"`
	KeyType           string `json:"key_type"`
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	PublicKey         string `json:"public_key"`
}

func (err *UnknownHostKeyError) Error() string {
	return fmt.Sprintf("ssh host key approval required for %s (%s)", err.Hostname, err.FingerprintSHA256)
}

func HostKeyCallback(path string) (ssh.HostKeyCallback, error) {
	path = filepath.Clean(path)
	if path == "." || path == "" {
		return nil, fmt.Errorf("known_hosts path is required")
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		knownHostsMu.Lock()
		defer knownHostsMu.Unlock()

		remote = knownHostsRemoteAddr(remote)
		if err := ensureKnownHostsFile(path); err != nil {
			return err
		}
		callback, err := knownhosts.New(path)
		if err != nil {
			return fmt.Errorf("load known_hosts: %w", err)
		}
		if err := callback(hostname, remote, key); err == nil {
			return nil
		} else {
			var keyErr *knownhosts.KeyError
			if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
				return fmt.Errorf("verify ssh host key: %w", err)
			}
		}

		return NewUnknownHostKeyError(hostname, key)
	}, nil
}

func NewUnknownHostKeyError(hostname string, key ssh.PublicKey) *UnknownHostKeyError {
	return &UnknownHostKeyError{
		Hostname:          hostname,
		KeyType:           key.Type(),
		FingerprintSHA256: HostKeyFingerprintSHA256(key),
		PublicKey:         base64.StdEncoding.EncodeToString(key.Marshal()),
	}
}

func ParseHostPublicKey(publicKey string) (ssh.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return nil, fmt.Errorf("decode host public key: %w", err)
	}
	key, err := ssh.ParsePublicKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse host public key: %w", err)
	}
	return key, nil
}

func HostKeyFingerprintSHA256(key ssh.PublicKey) string {
	return ssh.FingerprintSHA256(key)
}

func TrustHostKey(path string, hostname string, publicKey string) error {
	path = filepath.Clean(path)
	if path == "." || path == "" {
		return fmt.Errorf("known_hosts path is required")
	}
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	key, err := ParseHostPublicKey(publicKey)
	if err != nil {
		return err
	}

	knownHostsMu.Lock()
	defer knownHostsMu.Unlock()

	if err := ensureKnownHostsFile(path); err != nil {
		return err
	}
	callback, err := knownhosts.New(path)
	if err != nil {
		return fmt.Errorf("load known_hosts: %w", err)
	}
	if err := callback(hostname, knownHostsRemoteAddr(nil), key); err == nil {
		return nil
	} else {
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
			return fmt.Errorf("verify ssh host key: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, knownhosts.Line([]string{hostname}, key)); err != nil {
		return fmt.Errorf("append known_hosts: %w", err)
	}
	return nil
}

func knownHostsRemoteAddr(remote net.Addr) net.Addr {
	if remote != nil {
		return remote
	}
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}
}

func ensureKnownHostsFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create known_hosts directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	return file.Close()
}
