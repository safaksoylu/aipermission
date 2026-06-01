import { useCallback, useState } from "react";

export const idleActionState = { state: "idle", error: null, message: null };

export function useAsyncAction(initialState = idleActionState) {
  const [actionState, setActionState] = useState(initialState);

  const runAction = useCallback(async ({ pending = "saving", successMessage = null, action }) => {
    setActionState({ state: pending, error: null, message: null });
    try {
      const result = await action();
      const message = typeof successMessage === "function" ? successMessage(result) : successMessage;
      setActionState({ state: "idle", error: null, message });
      return result;
    } catch (error) {
      setActionState({ state: "error", error: error.message, message: null });
      return undefined;
    }
  }, []);

  const resetAction = useCallback(() => {
    setActionState(idleActionState);
  }, []);

  return { actionState, setActionState, runAction, resetAction };
}
