package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/aipermission/aipermission/backend/internal/execution"
	"github.com/aipermission/aipermission/backend/internal/servers"
)

func (s serverResourceHandlers) listServers(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	items, err := runtime.servers.List(r.Context())
	if err != nil {
		writeInternalError(w)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s serverResourceHandlers) createServer(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	var request servers.CreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	item, err := runtime.servers.Create(r.Context(), request)
	if err != nil {
		handleServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s serverResourceHandlers) getServer(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	item, err := runtime.servers.Get(r.Context(), id)
	if err != nil {
		handleServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s serverResourceHandlers) updateServer(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var request servers.UpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	item, err := runtime.servers.Update(r.Context(), id, request)
	if err != nil {
		handleServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s serverResourceHandlers) deleteServer(w http.ResponseWriter, r *http.Request) {
	runtime, ok := s.activeRuntimeOrLocked(w)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	removedKey := false
	if r.URL.Query().Get("remove_key") == "true" {
		server, privateKey, err := s.serverSSHMaterial(r.Context(), id)
		if err != nil {
			handleServerSSHMaterialError(w, err)
			return
		}
		sshKey, err := runtime.sshKeys.Get(r.Context(), server.SSHKeyID)
		if err != nil {
			writeInternalError(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		result, err := execution.RunCommand(ctx, s.executionTarget(server, privateKey), removeAuthorizedKeyCommand(sshKey.PublicKey))
		if err != nil {
			writeError(w, http.StatusBadGateway, "remote key uninstall failed")
			return
		}
		if result.ExitCode != 0 {
			message := strings.TrimSpace(result.Stderr + result.Stdout)
			if message == "" {
				message = "remote key uninstall failed"
			}
			writeError(w, http.StatusBadGateway, message)
			return
		}
		removedKey = true
	}

	if err := runtime.servers.Delete(r.Context(), id); err != nil {
		handleServerError(w, err)
		return
	}
	if removedKey {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "remote_key_removed": true})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func removeAuthorizedKeyCommand(publicKey string) string {
	blob := publicKeyBlob(publicKey)
	delimiter := "__AIPERMISSION_AUTHORIZED_KEY__"
	for strings.Contains(blob, "\n"+delimiter+"\n") {
		delimiter += "_X"
	}
	return `set -e
KEY_BLOB="$(cat <<'` + delimiter + `'
` + blob + `
` + delimiter + `
)"
if [ -z "$KEY_BLOB" ]; then
  echo "remote key uninstall failed: invalid public key" >&2
  exit 1
fi
mkdir -p ~/.ssh
touch ~/.ssh/authorized_keys
chmod 700 ~/.ssh
tmp="$HOME/.ssh/authorized_keys.aipermission.$$"
awk -v key_blob="$KEY_BLOB" '
BEGIN { removed = 0 }
{
  keep = 1
  for (i = 1; i <= NF; i++) {
    if ($i == key_blob) {
      keep = 0
      removed++
      break
    }
  }
  if (keep) print
}
END { print removed > "/dev/stderr" }
' ~/.ssh/authorized_keys 2>"$tmp.count" > "$tmp"
removed="$(cat "$tmp.count" 2>/dev/null || printf '0')"
rm -f "$tmp.count"
if [ "${removed:-0}" -eq 0 ]; then
  rm -f "$tmp"
  echo "remote key uninstall removed 0 authorized_keys entries" >&2
  exit 1
fi
cat "$tmp" > ~/.ssh/authorized_keys
rm -f "$tmp"
chmod 600 ~/.ssh/authorized_keys
printf 'aipermission_key_removed=%s\n' "$removed"`
}

func publicKeyBlob(publicKey string) string {
	fields := strings.Fields(publicKey)
	if len(fields) < 2 {
		return ""
	}
	return fields[1]
}
