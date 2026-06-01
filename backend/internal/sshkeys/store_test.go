package sshkeys

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbpkg "github.com/aipermission/aipermission/backend/internal/db"
	"github.com/aipermission/aipermission/backend/internal/vault"
)

func openSSHKeyTestStore(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	database, err := dbpkg.OpenEncrypted(filepath.Join(t.TempDir(), "test.db"), "test-password")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	secretVault, err := vault.New("gateway-secret")
	if err != nil {
		t.Fatalf("new vault: %v", err)
	}
	return database, NewStore(database, secretVault)
}

func TestInstallCommandShellQuotesPublicKey(t *testing.T) {
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA-test aipermission-main'$(touch /tmp/pwned)"
	command := InstallCommand(publicKey)

	if strings.Contains(command, `echo "`) {
		t.Fatalf("install command should not use double-quoted echo: %s", command)
	}
	if !strings.Contains(command, `printf '%s\n'`) {
		t.Fatalf("install command should use printf: %s", command)
	}
	if !strings.Contains(command, `'\''`) {
		t.Fatalf("install command should escape single quotes: %s", command)
	}
}

func TestSSHKeyStoreCreateListGetPrivateKeyAndDelete(t *testing.T) {
	ctx := context.Background()
	_, store := openSSHKeyTestStore(t)

	created, err := store.Create(ctx, CreateRequest{Name: " main key ", KeyType: ""})
	if err != nil {
		t.Fatalf("create ssh key: %v", err)
	}
	if created.Name != "main key" || created.KeyType != TypeED25519 {
		t.Fatalf("unexpected created key: %#v", created)
	}
	if !strings.HasPrefix(created.PublicKey, "ssh-ed25519 ") || !strings.Contains(created.PublicKey, "aipermission-main key") {
		t.Fatalf("unexpected public key: %s", created.PublicKey)
	}
	if created.InstallCommand == "" || !strings.Contains(created.InstallCommand, created.PublicKey) {
		t.Fatalf("install command should include public key")
	}

	private, err := store.GetPrivateKey(ctx, created.ID)
	if err != nil {
		t.Fatalf("get private key: %v", err)
	}
	if private.PrivateKey == "" || !strings.Contains(private.PrivateKey, "PRIVATE KEY") {
		t.Fatalf("private key was not decrypted: %#v", private)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected list: %#v", list)
	}
	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete key: %v", err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSSHKeyStoreCreatesRSAKey(t *testing.T) {
	ctx := context.Background()
	_, store := openSSHKeyTestStore(t)

	created, err := store.Create(ctx, CreateRequest{Name: "rsa", KeyType: TypeRSA})
	if err != nil {
		t.Fatalf("create rsa key: %v", err)
	}
	if created.KeyType != TypeRSA || !strings.HasPrefix(created.PublicKey, "ssh-rsa ") {
		t.Fatalf("unexpected rsa key: %#v", created)
	}
}

func TestSSHKeyStoreDuplicateNameReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	_, store := openSSHKeyTestStore(t)

	if _, err := store.Create(ctx, CreateRequest{Name: "main", KeyType: TypeED25519}); err != nil {
		t.Fatalf("create first key: %v", err)
	}
	if _, err := store.Create(ctx, CreateRequest{Name: "main", KeyType: TypeED25519}); !errors.As(err, new(ValidationError)) {
		t.Fatalf("expected duplicate key validation error, got %v", err)
	}
}

func TestSSHKeyStoreValidatesAndRefusesDeleteWhenInUse(t *testing.T) {
	ctx := context.Background()
	database, store := openSSHKeyTestStore(t)

	if ValidationError("bad key").Error() != "bad key" {
		t.Fatalf("validation error should return message")
	}
	if _, err := store.Create(ctx, CreateRequest{Name: "", KeyType: TypeED25519}); err == nil {
		t.Fatalf("expected empty name to fail")
	}
	if _, err := store.Create(ctx, CreateRequest{Name: "bad\nname", KeyType: TypeED25519}); err == nil {
		t.Fatalf("expected multiline name to fail")
	}
	if _, err := store.Create(ctx, CreateRequest{Name: strings.Repeat("a", 81), KeyType: TypeED25519}); err == nil {
		t.Fatalf("expected long name to fail")
	}
	if _, err := store.Create(ctx, CreateRequest{Name: "bad", KeyType: "dsa"}); err == nil {
		t.Fatalf("expected invalid key type to fail")
	}
	if _, err := store.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetPrivateKey(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for private key, got %v", err)
	}

	key, err := store.Create(ctx, CreateRequest{Name: "used", KeyType: TypeED25519})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO servers (name, host, port, username, ssh_key_id, description, created_at, updated_at)
		VALUES ('worker-1', '127.0.0.1', 22, 'root', ?, '', ?, ?)`,
		key.ID,
		now,
		now,
	); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	if err := store.Delete(ctx, key.ID); err == nil {
		t.Fatalf("expected delete to fail while key is in use")
	}
}
