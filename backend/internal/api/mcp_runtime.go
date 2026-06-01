package api

import (
	"net/http"
	"time"
)

type mcpRuntimeResponse struct {
	Enabled      bool   `json:"enabled"`
	StartEnabled bool   `json:"start_enabled"`
	UpdatedAt    string `json:"updated_at"`
}

type updateMCPRuntimeRequest struct {
	Enabled bool `json:"enabled"`
}

func (s mcpHandlers) getMCPRuntime(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	settings, err := readSecuritySettings(r.Context(), runtime)
	if err != nil {
		writeInternalError(w)
		return
	}
	writeJSON(w, http.StatusOK, mcpRuntimeResponse{
		Enabled:      runtime.isMCPStarted(),
		StartEnabled: settings.MCPStartEnabled,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s mcpHandlers) updateMCPRuntime(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	var request updateMCPRuntimeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	runtime.setMCPStarted(request.Enabled)
	action := "mcp.runtime.stopped"
	if request.Enabled {
		action = "mcp.runtime.started"
	}
	s.writeAudit(r.Context(), runtime, "user", nil, 0, action, map[string]any{
		"enabled": request.Enabled,
	})
	settings, err := readSecuritySettings(r.Context(), runtime)
	if err != nil {
		writeInternalError(w)
		return
	}
	writeJSON(w, http.StatusOK, mcpRuntimeResponse{
		Enabled:      runtime.isMCPStarted(),
		StartEnabled: settings.MCPStartEnabled,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (runtime *databaseRuntime) isMCPStarted() bool {
	runtime.mcpMu.RLock()
	defer runtime.mcpMu.RUnlock()
	return runtime.mcpStarted
}

func (runtime *databaseRuntime) setMCPStarted(enabled bool) {
	runtime.mcpMu.Lock()
	defer runtime.mcpMu.Unlock()
	runtime.mcpStarted = enabled
}

func (s *Server) rejectStoppedMCP(w http.ResponseWriter, runtime *databaseRuntime) bool {
	if runtime.isMCPStarted() {
		return false
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "stopped",
		"error":  "MCP execution is stopped in the local gateway. Start MCP from the web UI before running commands.",
	})
	return true
}
