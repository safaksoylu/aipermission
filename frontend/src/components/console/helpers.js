export const emptySession = {
  transcript: "",
  status: "idle",
  error: null,
};

export function isUnreadMessage(message) {
  return message.direction === "ai_to_user" && !message.consumed_at;
}

export function isLiveConsoleSession(session) {
  return session?.status === "connecting" || session?.status === "connected";
}

export function latestSessionForServer(sessions, serverID) {
  return sessions.find((session) => Number(session.server_id) === Number(serverID)) || null;
}
