package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aipermission/aipermission/backend/internal/tokens"
)

type messageRecord struct {
	ID         int64   `json:"id"`
	TokenID    int64   `json:"token_id"`
	TokenName  string  `json:"token_name,omitempty"`
	ServerID   *int64  `json:"server_id,omitempty"`
	ServerName string  `json:"server_name,omitempty"`
	SessionID  *int64  `json:"session_id,omitempty"`
	Direction  string  `json:"direction"`
	Message    string  `json:"message"`
	ConsumedAt *string `json:"consumed_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type createMessageRequest struct {
	TokenID   int64  `json:"token_id"`
	ServerID  *int64 `json:"server_id"`
	SessionID *int64 `json:"session_id"`
	Direction string `json:"direction"`
	Message   string `json:"message"`
}

type markMessagesReadRequest struct {
	ServerID int64 `json:"server_id"`
}

func (s messageHandlers) listMessages(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	filter := messageFilter{Direction: strings.TrimSpace(r.URL.Query().Get("direction"))}
	if rawServerID := strings.TrimSpace(r.URL.Query().Get("server_id")); rawServerID != "" {
		id, ok := parseInt64Query(w, rawServerID, "server_id")
		if !ok {
			return
		}
		filter.ServerID = id
	}
	items, err := s.listMessageRecords(r.Context(), runtime, filter)
	if err != nil {
		writeInternalError(w)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s messageHandlers) createMessage(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	var request createMessageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	item, err := s.insertMessage(r.Context(), runtime, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s messageHandlers) markMessagesRead(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	var request markMessagesReadRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if request.ServerID < 1 {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := runtime.database.ExecContext(r.Context(), `
		UPDATE message_queue
		SET consumed_at = ?
		WHERE direction = 'ai_to_user' AND consumed_at IS NULL AND server_id = ?`,
		now,
		request.ServerID,
	)
	if err != nil {
		writeInternalError(w)
		return
	}
	affected, _ := result.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]any{"status": "read", "count": affected})
}

func (s mcpHandlers) mcpCreateMessage(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.authenticateMCP(w, r)
	if !ok {
		return
	}
	if s.rejectStoppedMCP(w, auth.runtime) {
		return
	}
	var request createMessageRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	request.TokenID = auth.TokenID
	request.Direction = "ai_to_user"
	if request.SessionID != nil {
		serverID, err := consoleSessionServerID(r.Context(), auth.runtime, *request.SessionID)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "session_id not found")
			return
		}
		if err != nil {
			writeInternalError(w)
			return
		}
		if request.ServerID != nil && *request.ServerID != serverID {
			writeError(w, http.StatusBadRequest, "session_id does not belong to server_id")
			return
		}
		request.ServerID = &serverID
	}
	if request.ServerID != nil {
		_, rule, allowed, err := s.mcpPermission(r.Context(), auth.runtime, auth.TokenID, *request.ServerID)
		if err != nil {
			writeInternalError(w)
			return
		}
		if !allowed || rule == tokens.RuleBlocked {
			writeError(w, http.StatusForbidden, "token is not allowed to send messages for this server")
			return
		}
	}
	item, err := s.insertMessage(r.Context(), auth.runtime, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type messageFilter struct {
	TokenID   int64
	ServerID  int64
	Direction string
}

func (s *Server) insertMessage(ctx context.Context, runtime *databaseRuntime, request createMessageRequest) (messageRecord, error) {
	request.Message = strings.TrimSpace(request.Message)
	request.Direction = strings.TrimSpace(request.Direction)
	if request.Direction == "" {
		request.Direction = "user_to_ai"
	}
	if request.Direction != "user_to_ai" && request.Direction != "ai_to_user" {
		return messageRecord{}, errors.New("direction must be user_to_ai or ai_to_user")
	}
	if request.TokenID < 1 {
		return messageRecord{}, errors.New("token_id is required")
	}
	if request.Message == "" {
		return messageRecord{}, errors.New("message is required")
	}
	if err := validateTextLimit("message", request.Message, maxMessageBytes); err != nil {
		return messageRecord{}, err
	}
	if err := validateMessageScope(ctx, runtime, &request); err != nil {
		return messageRecord{}, err
	}
	request.Message = s.redactForPersistence(ctx, runtime, request.Message)

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := runtime.database.ExecContext(ctx, `
		INSERT INTO message_queue (token_id, server_id, session_id, direction, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		request.TokenID,
		nullableInt64(request.ServerID),
		nullableInt64(request.SessionID),
		request.Direction,
		request.Message,
		now,
	)
	if err != nil {
		return messageRecord{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return messageRecord{}, err
	}
	return s.getMessageRecord(ctx, runtime, id)
}

func validateMessageScope(ctx context.Context, runtime *databaseRuntime, request *createMessageRequest) error {
	var tokenExists int
	err := runtime.database.QueryRowContext(ctx, `SELECT 1 FROM api_tokens WHERE id = ?`, request.TokenID).Scan(&tokenExists)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("token_id does not exist")
	}
	if err != nil {
		return err
	}

	if request.SessionID != nil {
		var sessionServerID int64
		err := runtime.database.QueryRowContext(ctx, `SELECT server_id FROM console_sessions WHERE id = ?`, *request.SessionID).Scan(&sessionServerID)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("session_id does not exist")
		}
		if err != nil {
			return err
		}
		if request.ServerID != nil && *request.ServerID != sessionServerID {
			return errors.New("session_id does not belong to server_id")
		}
		request.ServerID = &sessionServerID
	}

	if request.ServerID != nil {
		var serverExists int
		err := runtime.database.QueryRowContext(ctx, `SELECT 1 FROM servers WHERE id = ?`, *request.ServerID).Scan(&serverExists)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("server_id does not exist")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) consumeNextUserMessage(ctx context.Context, runtime *databaseRuntime, tokenID int64, serverID int64, sessionID int64) (*string, error) {
	tx, err := runtime.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
			SELECT id, message
			FROM message_queue
			WHERE token_id = ? AND direction = 'user_to_ai' AND consumed_at IS NULL
				AND ((? > 0 AND server_id = ?) OR server_id IS NULL)
				AND ((? > 0 AND session_id = ?) OR session_id IS NULL)
			ORDER BY
				CASE
					WHEN ? > 0 AND session_id = ? THEN 0
					WHEN server_id = ? THEN 1
					ELSE 2
				END,
				created_at ASC,
				id ASC
			LIMIT 1`,
		tokenID,
		serverID,
		serverID,
		sessionID,
		sessionID,
		sessionID,
		sessionID,
		serverID,
	)
	var id int64
	var message string
	if err := row.Scan(&id, &message); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := tx.ExecContext(ctx, `
		UPDATE message_queue
		SET consumed_at = ?
		WHERE id = ? AND consumed_at IS NULL`,
		now,
		id,
	)
	if err != nil {
		return nil, err
	}
	if err := requireAffected(result); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &message, nil
}

func (s *Server) nextUserMessage(ctx context.Context, runtime *databaseRuntime, tokenID int64, serverID int64, sessionID int64) (messageRecord, error) {
	row := runtime.database.QueryRowContext(ctx, messageSelectSQL()+`
			WHERE mq.token_id = ? AND mq.direction = 'user_to_ai' AND mq.consumed_at IS NULL
				AND ((? > 0 AND mq.server_id = ?) OR mq.server_id IS NULL)
				AND ((? > 0 AND mq.session_id = ?) OR mq.session_id IS NULL)
			ORDER BY
				CASE
					WHEN ? > 0 AND mq.session_id = ? THEN 0
					WHEN mq.server_id = ? THEN 1
					ELSE 2
				END,
				mq.created_at ASC
			LIMIT 1`,
		tokenID,
		serverID,
		serverID,
		sessionID,
		sessionID,
		sessionID,
		sessionID,
		serverID,
	)
	return scanMessage(row)
}

func consoleSessionServerID(ctx context.Context, runtime *databaseRuntime, sessionID int64) (int64, error) {
	var serverID int64
	err := runtime.database.QueryRowContext(ctx, `SELECT server_id FROM console_sessions WHERE id = ?`, sessionID).Scan(&serverID)
	return serverID, err
}

func (s *Server) getMessageRecord(ctx context.Context, runtime *databaseRuntime, id int64) (messageRecord, error) {
	row := runtime.database.QueryRowContext(ctx, messageSelectSQL()+` WHERE mq.id = ?`, id)
	return scanMessage(row)
}

func (s *Server) listMessageRecords(ctx context.Context, runtime *databaseRuntime, filter messageFilter) ([]messageRecord, error) {
	where := []string{"(? = 0 OR mq.token_id = ?)", "(? = 0 OR mq.server_id = ?)", "(? = '' OR mq.direction = ?)"}
	args := []any{filter.TokenID, filter.TokenID, filter.ServerID, filter.ServerID, filter.Direction, filter.Direction}
	rows, err := runtime.database.QueryContext(ctx, messageSelectSQL()+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY mq.created_at DESC
		LIMIT 100`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []messageRecord{}
	for rows.Next() {
		item, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func messageSelectSQL() string {
	return `
		SELECT mq.id, mq.token_id, COALESCE(tok.name, ''), mq.server_id, COALESCE(srv.name, ''), mq.session_id,
		       mq.direction, mq.message, mq.consumed_at, mq.created_at
		FROM message_queue mq
		JOIN api_tokens tok ON tok.id = mq.token_id
		LEFT JOIN servers srv ON srv.id = mq.server_id`
}

func scanMessage(scanner interface {
	Scan(dest ...any) error
}) (messageRecord, error) {
	var item messageRecord
	var serverID sql.NullInt64
	var sessionID sql.NullInt64
	var consumedAt sql.NullString
	err := scanner.Scan(
		&item.ID,
		&item.TokenID,
		&item.TokenName,
		&serverID,
		&item.ServerName,
		&sessionID,
		&item.Direction,
		&item.Message,
		&consumedAt,
		&item.CreatedAt,
	)
	if err != nil {
		return messageRecord{}, err
	}
	if serverID.Valid {
		value := serverID.Int64
		item.ServerID = &value
	}
	if sessionID.Valid {
		value := sessionID.Int64
		item.SessionID = &value
	}
	if consumedAt.Valid {
		item.ConsumedAt = stringPtr(consumedAt.String)
	}
	return item, nil
}

func nullableInt64(value *int64) any {
	if value == nil || *value == 0 {
		return nil
	}
	return *value
}
