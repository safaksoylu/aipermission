package console

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	dbpkg "github.com/aipermission/aipermission/backend/internal/db"
	"github.com/gorilla/websocket"
)

func TestManagedConsoleSessionCommandResultParsing(t *testing.T) {
	session := &managedConsoleSession{
		status:        "connected",
		rawTranscript: "prompt\ncommand output\n__AIPERMISSION_EXIT_1:42\nnext prompt",
	}

	output, exitCode, completed, err := session.checkCommandResult(0, "__AIPERMISSION_EXIT_1")
	if err != nil {
		t.Fatalf("check result: %v", err)
	}
	if !completed || exitCode != 42 || output != "prompt\ncommand output" {
		t.Fatalf("unexpected result completed=%v exit=%d output=%q", completed, exitCode, output)
	}
}

func TestManagedConsoleSessionReportsClosedBeforeCommandMarker(t *testing.T) {
	session := &managedConsoleSession{
		status:        "closed",
		rawTranscript: "partial output",
	}

	output, _, completed, err := session.checkCommandResult(0, "missing")
	if err == nil || completed || output != "partial output" {
		t.Fatalf("expected closed error with partial output, completed=%v output=%q err=%v", completed, output, err)
	}
}

func TestConsoleTranscriptLimitKeepsTail(t *testing.T) {
	value := strings.Repeat("a", maxConsoleTranscriptLength+20)
	limited := limitConsoleTranscript(value)
	if len(limited) != maxConsoleTranscriptLength {
		t.Fatalf("unexpected limited length: %d", len(limited))
	}
	if limited != value[20:] {
		t.Fatalf("transcript should keep the newest tail")
	}
}

func TestConsoleSessionManagerActiveForServerUsesNewestLiveSession(t *testing.T) {
	manager := &Manager{
		sessions: map[int64]*managedConsoleSession{
			1: {id: 1, serverID: 10, status: "connected"},
			2: {id: 2, serverID: 10, status: "closed"},
			3: {id: 3, serverID: 10, status: "connected"},
			4: {id: 4, serverID: 11, status: "connected"},
		},
	}

	selected := manager.activeForServer(10)
	if selected == nil || selected.id != 3 {
		t.Fatalf("expected newest live session 3, got %#v", selected)
	}
}

func TestConsoleSessionManagerEnforcesActiveSessionLimit(t *testing.T) {
	manager := &Manager{sessions: map[int64]*managedConsoleSession{}}
	for i := 0; i < maxActiveConsoleSessions; i++ {
		manager.sessions[int64(i+1)] = &managedConsoleSession{id: int64(i + 1), status: "connected"}
	}

	if _, err := manager.Create(context.Background(), CreateRequest{ServerID: 1}); !errors.Is(err, ErrSessionLimit) {
		t.Fatalf("expected session limit error, got %v", err)
	}
}

func TestManagedConsoleSessionEnforcesClientLimit(t *testing.T) {
	session := &managedConsoleSession{clients: map[*websocket.Conn]*sync.Mutex{}}
	for i := 0; i < maxConsoleClientsPerSession; i++ {
		if _, err := session.addClient(&websocket.Conn{}); err != nil {
			t.Fatalf("add client %d: %v", i, err)
		}
	}
	if _, err := session.addClient(&websocket.Conn{}); !errors.Is(err, ErrClientLimit) {
		t.Fatalf("expected client limit error, got %v", err)
	}
}

func TestConsoleSessionInputLimit(t *testing.T) {
	manager := &Manager{sessions: map[int64]*managedConsoleSession{}}
	if err := manager.Input(context.Background(), 1, strings.Repeat("x", maxConsoleInputBytes+1)); !errors.Is(err, ErrInputTooLarge) {
		t.Fatalf("expected input limit error, got %v", err)
	}
}

func TestManagedConsoleSessionExecRejectsConcurrentAutomationCommand(t *testing.T) {
	session := &managedConsoleSession{
		id:            7,
		status:        "connected",
		rawTranscript: "prompt\nlong command output\n",
		activeExec: &consoleSessionActiveExec{
			Command:     "sleep 60",
			Marker:      "__AIPERMISSION_EXIT_ACTIVE__",
			StartOffset: 0,
			Started:     time.Now().Add(-time.Second),
		},
	}

	result, err := session.execCommand(context.Background(), "docker ps")
	if !errors.Is(err, ErrCommandActive) {
		t.Fatalf("expected active command error, got %v", err)
	}
	if !result.Running || result.Command != "sleep 60" || result.SessionID != 7 {
		t.Fatalf("expected active command metadata, got %#v", result)
	}
	if active := session.activeCommand(); active == nil || active.Command != "sleep 60" {
		t.Fatalf("concurrent exec must not replace active command: %#v", active)
	}
}

func TestManagedConsoleSessionExecRejectsCompletedActiveCommandUntilFinalized(t *testing.T) {
	session := &managedConsoleSession{
		id:            7,
		status:        "connected",
		rawTranscript: "prompt\nold output\n__AIPERMISSION_EXIT_ACTIVE__:0\n",
		activeExec: &consoleSessionActiveExec{
			Command:     "apt update",
			Marker:      "__AIPERMISSION_EXIT_ACTIVE__",
			StartOffset: 0,
			Started:     time.Now().Add(-time.Second),
		},
	}

	result, err := session.execCommand(context.Background(), "docker ps")
	if !errors.Is(err, ErrCommandActive) {
		t.Fatalf("expected active command error, got %v", err)
	}
	if result.Command != "apt update" || result.ExitCode != 0 {
		t.Fatalf("expected previous command metadata, got %#v", result)
	}
	if active := session.activeCommand(); active == nil || active.Command != "apt update" {
		t.Fatalf("completed active command should remain for background finalizer: %#v", active)
	}
}

func TestManagedConsoleSessionWaitActiveDoesNotBlockConcurrentExec(t *testing.T) {
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	defer sessionCancel()
	session := &managedConsoleSession{
		id:            7,
		ctx:           sessionCtx,
		status:        "connected",
		rawTranscript: "prompt\nlong command output\n",
		activeExec: &consoleSessionActiveExec{
			Command:     "sleep 60",
			Marker:      "__AIPERMISSION_EXIT_ACTIVE__",
			StartOffset: 0,
			Started:     time.Now().Add(-time.Second),
		},
	}

	waitCtx, cancelWait := context.WithCancel(context.Background())
	defer cancelWait()
	waitStarted := make(chan struct{})
	var once sync.Once
	go func() {
		once.Do(func() { close(waitStarted) })
		_, _ = session.waitActiveCommand(waitCtx)
	}()
	<-waitStarted
	time.Sleep(25 * time.Millisecond)

	execCtx, cancelExec := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelExec()
	result, err := session.execCommand(execCtx, "docker ps")
	if !errors.Is(err, ErrCommandActive) {
		t.Fatalf("expected active command error, got %v", err)
	}
	if !result.Running || result.Command != "sleep 60" {
		t.Fatalf("expected active command metadata, got %#v", result)
	}
}

func TestConsoleExecPayloadAvoidsBase64BashAndMktemp(t *testing.T) {
	payload := consoleExecPayload("printf 'hello\\n'\n", "__AIPERMISSION_EXIT_TEST__")
	for _, forbidden := range []string{"base64", "mktemp", "bash "} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("payload should not depend on %q: %s", forbidden, payload)
		}
	}
	if !strings.Contains(payload, "/bin/sh <<'__AIPERMISSION_EXIT_TEST___SCRIPT'") {
		t.Fatalf("payload should run command through a quoted /bin/sh heredoc: %s", payload)
	}
	if !strings.Contains(payload, "__AIPERMISSION_EXIT_TEST__:%s") {
		t.Fatalf("payload should emit the exit marker: %s", payload)
	}
	if !strings.Contains(payload, ") </dev/null") {
		t.Fatalf("payload should keep command stdin from consuming the heredoc: %s", payload)
	}
	if !strings.Contains(payload, "stty sane 2>/dev/null || stty echo icanon opost 2>/dev/null || true") {
		t.Fatalf("payload should restore terminal input mode before completion marker: %s", payload)
	}
}

func TestCleanConsoleDisplayOutputRemovesInternalExecNoise(t *testing.T) {
	input := "root@worker:~# __aipermission_saved_ps2=${PS2-}\r\n" +
		"root@worker:~# PS2=\r\n" +
		"root@worker:~# stty -echo\r\n" +
		"root@worker:~# stty sane 2>/dev/null || stty echo icanon opost 2>/dev/null || true\r\n" +
		"\r\n\r\n" +
		"--- images before ---\r\n" +
		"nginx alpine\r\n" +
		"\r\n" +
		"__AIPERMISSION_EXIT_1_2__:0\r\n" +
		"root@worker:~# \r\n" +
		"\r\n\r\n" +
		"--- after prompt ---\r\n"

	output := cleanConsoleDisplayOutput(input, false)
	for _, forbidden := range []string{"root@worker", "__aipermission", "PS2=", "stty -echo", "stty sane", "__AIPERMISSION_EXIT", "\r\n\r\n"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("display output should remove %q noise: %q", forbidden, output)
		}
	}
	if !strings.Contains(output, "--- images before ---") || !strings.Contains(output, "nginx alpine") || !strings.Contains(output, "--- after prompt ---") {
		t.Fatalf("display output should keep useful lines: %q", output)
	}
}

func TestAppendOutputKeepsManualPTYCharacters(t *testing.T) {
	session := &managedConsoleSession{}
	session.appendOutput("cd /opt/aiperm demo")
	session.appendOutput("\b\b")
	session.appendOutput("data")

	if session.transcript != "cd /opt/aiperm demo\b\bdata" {
		t.Fatalf("manual output should stay raw, got %q", session.transcript)
	}
}

func TestAppendOutputCleansOnlyDuringAutomation(t *testing.T) {
	session := &managedConsoleSession{
		activeExec: &consoleSessionActiveExec{Marker: "__AIPERMISSION_EXIT_1"},
	}
	session.appendOutput("root@worker:~# PS2=\r\n\r\nuseful\r\n__AIPERMISSION_EXIT_1:0\r\n")

	if strings.Contains(session.transcript, "PS2=") || strings.Contains(session.transcript, "__AIPERMISSION_EXIT") || strings.Contains(session.transcript, "\r\n\r\n") {
		t.Fatalf("automation output should be cleaned for display: %q", session.transcript)
	}
	if !strings.Contains(session.rawTranscript, "__AIPERMISSION_EXIT_1:0") {
		t.Fatalf("raw transcript should keep marker for parsing: %q", session.rawTranscript)
	}
	if !strings.Contains(session.transcript, "useful") {
		t.Fatalf("display transcript should keep useful output: %q", session.transcript)
	}
}

func TestAppendOutputKeepsFinalPromptAfterAutomationCompletes(t *testing.T) {
	session := &managedConsoleSession{
		activeExec: &consoleSessionActiveExec{Marker: "__AIPERMISSION_EXIT_1"},
	}
	session.appendOutput("useful\r\n__AIPERMISSION_EXIT_1:0\r\n")
	session.clearActiveCommand("__AIPERMISSION_EXIT_1")
	session.appendOutput("root@worker:~# \r\n\r\n\r\n")

	if !strings.Contains(session.transcript, "root@worker:~#") {
		t.Fatalf("post-automation prompt should stay visible: %q", session.transcript)
	}
	if strings.Contains(session.transcript, "\r\n\r\n") {
		t.Fatalf("post-automation blank tail should be filtered: %q", session.transcript)
	}
	if !strings.Contains(session.rawTranscript, "root@worker:~#") {
		t.Fatalf("raw transcript should keep prompt tail for diagnostics: %q", session.rawTranscript)
	}
}

func TestAppendOutputKeepsPromptAfterMarkerWhileAutomationStillActive(t *testing.T) {
	session := &managedConsoleSession{
		activeExec: &consoleSessionActiveExec{Marker: "__AIPERMISSION_EXIT_1"},
	}
	session.appendOutput("useful\r\n__AIPERMISSION_EXIT_1:0\r\n")
	session.appendOutput("root@worker:~# ")

	if !strings.Contains(session.transcript, "root@worker:~#") {
		t.Fatalf("prompt after marker should stay visible even before active command is cleared: %q", session.transcript)
	}
	if strings.Contains(session.transcript, "__AIPERMISSION_EXIT") {
		t.Fatalf("display transcript should still hide internal marker: %q", session.transcript)
	}
}

func TestAppendOutputKeepsManualOutputAfterFilterWindow(t *testing.T) {
	session := &managedConsoleSession{
		filterUntil: time.Now().Add(-time.Second),
	}
	session.appendOutput("manual line\r\n\r\n")

	if session.transcript != "manual line\r\n\r\n" {
		t.Fatalf("manual output after filter window should stay raw: %q", session.transcript)
	}
}

func TestFormatAutomationCommandShowsCommandLines(t *testing.T) {
	output := formatAutomationCommand("set -e\n\ndocker ps\n")
	if !strings.Contains(output, "[AI command]") ||
		!strings.Contains(output, "$ set -e") ||
		!strings.Contains(output, "$ docker ps") {
		t.Fatalf("automation command should be visible in display transcript: %q", output)
	}
	if strings.Contains(output, "$ \r\n") || strings.HasPrefix(output, "\r\n") {
		t.Fatalf("automation command should avoid blank command rows and leading blank lines: %q", output)
	}
}

func TestAppendDisplayOutputSeparatesAutomationCommandFromPrompt(t *testing.T) {
	session := &managedConsoleSession{
		transcript: "root@worker:~# ",
	}
	session.appendDisplayOutput(formatAutomationCommand("pwd"))

	if !strings.Contains(session.transcript, "root@worker:~# \r\n[AI command]") {
		t.Fatalf("automation command should start after the current prompt line: %q", session.transcript)
	}
}

func TestConsoleSessionManagerCreateValidationAndCloseInactive(t *testing.T) {
	database, err := dbpkg.OpenEncrypted(filepath.Join(t.TempDir(), "console.db"), "ConsolePassword123")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	manager := NewManager(database, nil, "", nil)

	if _, err := manager.Create(context.Background(), CreateRequest{}); err == nil {
		t.Fatalf("expected missing server id to fail")
	}
	if err := manager.Close(context.Background(), 999); err != nil {
		t.Fatalf("closing inactive/missing session should be idempotent: %v", err)
	}
}

func TestConsoleSessionManagerListGetAndCloseServer(t *testing.T) {
	database, err := dbpkg.OpenEncrypted(filepath.Join(t.TempDir(), "console.db"), "ConsolePassword123")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := database.Exec(`
		INSERT INTO servers (name, host, port, username, ssh_key_id, created_at, updated_at)
		VALUES ('worker-1', '127.0.0.1', 22, 'root', 0, ?, ?)`,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert server: %v", err)
	}
	serverID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read server id: %v", err)
	}
	sessionResult, err := database.Exec(`
		INSERT INTO console_sessions (server_id, name, status, transcript, cols, rows, created_at, updated_at)
		VALUES (?, 'manual', 'connected', 'hello', 120, 32, ?, ?)`,
		serverID,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert console session: %v", err)
	}
	sessionID, err := sessionResult.LastInsertId()
	if err != nil {
		t.Fatalf("read session id: %v", err)
	}

	manager := NewManager(database, nil, "", nil)
	items, err := manager.List(context.Background(), serverID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(items) != 1 || items[0].ID != sessionID || items[0].ServerName != "worker-1" || items[0].Transcript != "hello" {
		t.Fatalf("unexpected session list: %#v", items)
	}
	item, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if item.ID != sessionID || item.Status != "connected" {
		t.Fatalf("unexpected session: %#v", item)
	}
	if err := manager.CloseServer(context.Background(), serverID); err != nil {
		t.Fatalf("close server sessions: %v", err)
	}
	item, err = manager.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("get closed session: %v", err)
	}
	if item.Status != "closed" || item.ClosedAt == nil {
		t.Fatalf("expected closed session, got %#v", item)
	}
}

func TestConsoleTranscriptPersistsAppendOnlyChunksAndBoundedSnapshot(t *testing.T) {
	database, err := dbpkg.OpenEncrypted(filepath.Join(t.TempDir(), "console.db"), "ConsolePassword123")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := database.Exec(`
		INSERT INTO servers (name, host, port, username, ssh_key_id, created_at, updated_at)
		VALUES ('worker-1', '127.0.0.1', 22, 'root', 0, ?, ?)`,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert server: %v", err)
	}
	serverID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read server id: %v", err)
	}
	sessionResult, err := database.Exec(`
		INSERT INTO console_sessions (server_id, name, status, cols, rows, created_at, updated_at)
		VALUES (?, 'manual', 'connected', 120, 32, ?, ?)`,
		serverID,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert console session: %v", err)
	}
	sessionID, err := sessionResult.LastInsertId()
	if err != nil {
		t.Fatalf("read session id: %v", err)
	}

	manager := NewManager(database, nil, "", nil)
	output := strings.Repeat("a", maxConsoleSnapshotLength+10) + strings.Repeat("b", maxConsoleChunkLength+5)
	session := &managedConsoleSession{
		id:      sessionID,
		manager: manager,
		status:  "connected",
		clients: map[*websocket.Conn]*sync.Mutex{},
	}
	session.appendDisplayOutput(output)
	session.flushTranscript()

	var snapshot string
	if err := database.QueryRow(`SELECT transcript FROM console_sessions WHERE id = ?`, sessionID).Scan(&snapshot); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if len(snapshot) != maxConsoleSnapshotLength {
		t.Fatalf("expected bounded snapshot length %d, got %d", maxConsoleSnapshotLength, len(snapshot))
	}
	if !strings.HasSuffix(snapshot, strings.Repeat("b", maxConsoleChunkLength+5)) {
		t.Fatalf("snapshot should keep the newest transcript tail")
	}

	var chunks int
	if err := database.QueryRow(`SELECT COUNT(*) FROM console_session_chunks WHERE session_id = ?`, sessionID).Scan(&chunks); err != nil {
		t.Fatalf("count transcript chunks: %v", err)
	}
	if chunks < 2 {
		t.Fatalf("expected transcript to be split across chunks, got %d", chunks)
	}

	record, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if record.Transcript != output {
		t.Fatalf("get should reconstruct transcript tail from chunks")
	}

	if _, err := database.Exec(`DELETE FROM console_sessions WHERE id = ?`, sessionID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM console_session_chunks WHERE session_id = ?`, sessionID).Scan(&chunks); err != nil {
		t.Fatalf("count chunks after cascade: %v", err)
	}
	if chunks != 0 {
		t.Fatalf("expected console transcript chunks to cascade delete, got %d", chunks)
	}
}
