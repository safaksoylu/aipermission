package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveAuthorizedKeyCommandRemovesByPublicKeyBlob(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("create .ssh: %v", err)
	}
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTKEY aipermission-main"
	other := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOTHER other"
	authorizedKeys := strings.Join([]string{
		`from="127.0.0.1" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTKEY custom-comment`,
		other,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(sshDir, "authorized_keys"), []byte(authorizedKeys), 0o600); err != nil {
		t.Fatalf("write authorized_keys: %v", err)
	}

	command := exec.Command("sh", "-c", removeAuthorizedKeyCommand(key))
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("remove command failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "aipermission_key_removed=1") {
		t.Fatalf("expected removal count, got %s", output)
	}
	updated, err := os.ReadFile(filepath.Join(sshDir, "authorized_keys"))
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if strings.Contains(string(updated), "AAAAITESTKEY") {
		t.Fatalf("target key blob should be removed: %s", updated)
	}
	if !strings.Contains(string(updated), "AAAAIOTHER") {
		t.Fatalf("other key should remain: %s", updated)
	}
}

func TestRemoveAuthorizedKeyCommandFailsWhenNoEntryRemoved(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatalf("create .ssh: %v", err)
	}
	command := exec.Command("sh", "-c", removeAuthorizedKeyCommand("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMISSING aipermission-main"))
	command.Env = append(os.Environ(), "HOME="+home)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected command to fail when no key is removed, got %s", output)
	}
	if !strings.Contains(string(output), "removed 0") {
		t.Fatalf("expected removed 0 message, got %s", output)
	}
}
