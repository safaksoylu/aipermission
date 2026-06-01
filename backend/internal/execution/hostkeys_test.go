package execution

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestHostKeyRequiresExplicitTrustBeforeFirstUse(t *testing.T) {
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	hostKey, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatalf("ssh public key: %v", err)
	}

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	callback, err := HostKeyCallback(knownHostsPath)
	if err != nil {
		t.Fatalf("host key callback: %v", err)
	}

	var unknown *UnknownHostKeyError
	if err := callback("[example.test]:2222", nil, hostKey); !errors.As(err, &unknown) {
		t.Fatalf("expected unknown host key error, got %T: %v", err, err)
	}
	if unknown.FingerprintSHA256 != HostKeyFingerprintSHA256(hostKey) {
		t.Fatalf("unexpected fingerprint: %q", unknown.FingerprintSHA256)
	}

	if err := TrustHostKey(knownHostsPath, "[example.test]:2222", unknown.PublicKey); err != nil {
		t.Fatalf("trust host key: %v", err)
	}
	if err := callback("[example.test]:2222", nil, hostKey); err != nil {
		t.Fatalf("trusted host key should pass: %v", err)
	}
}
