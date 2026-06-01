package api

import (
	"context"
	"testing"
	"time"

	dbpkg "github.com/aipermission/aipermission/backend/internal/db"
)

func TestCommandRequestPendingTransitionIsSingleUse(t *testing.T) {
	database, err := dbpkg.OpenEncrypted(t.TempDir()+"/test.db", "test-password")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO servers (name, host, port, username, ssh_key_id, description, created_at, updated_at)
		VALUES ('core-1', '127.0.0.1', 22, 'root', 1, '', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	result, err := database.Exec(`
		INSERT INTO command_requests (server_id, command, reason, status, created_at)
		VALUES (1, 'ls', 'test', 'pending_approval', ?)`,
		now,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	server := &Server{}
	runtime := &databaseRuntime{database: database}
	if err := server.markCommandRequestRunning(context.Background(), runtime, id); err != nil {
		t.Fatalf("first transition failed: %v", err)
	}
	if err := server.markCommandRequestRunning(context.Background(), runtime, id); err != errCommandRequestNotPending {
		t.Fatalf("expected second transition to fail with errCommandRequestNotPending, got %v", err)
	}
}

func TestDeclinePendingTransitionIsSingleUse(t *testing.T) {
	database, err := dbpkg.OpenEncrypted(t.TempDir()+"/test.db", "test-password")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`
		INSERT INTO servers (name, host, port, username, ssh_key_id, description, created_at, updated_at)
		VALUES ('core-1', '127.0.0.1', 22, 'root', 1, '', ?, ?)`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	result, err := database.Exec(`
		INSERT INTO command_requests (server_id, command, reason, status, created_at)
		VALUES (1, 'ls', 'test', 'pending_approval', ?)`,
		now,
	)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	server := &Server{}
	runtime := &databaseRuntime{database: database}
	if err := server.declineCommandRequest(context.Background(), runtime, id, "no"); err != nil {
		t.Fatalf("first transition failed: %v", err)
	}
	var errorText string
	if err := database.QueryRow(`SELECT error FROM command_requests WHERE id = ?`, id).Scan(&errorText); err != nil {
		t.Fatalf("read declined error: %v", err)
	}
	if errorText != "User declined the command" {
		t.Fatalf("unexpected decline error: %q", errorText)
	}
	if err := server.declineCommandRequest(context.Background(), runtime, id, "no"); err != errCommandRequestNotPending {
		t.Fatalf("expected second transition to fail with errCommandRequestNotPending, got %v", err)
	}
}
