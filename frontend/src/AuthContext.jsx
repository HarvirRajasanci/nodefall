import { useState } from "react";
import { AuthContext, STORAGE_KEY, decodeUserID } from "./authContext";

export function AuthProvider({ children }) {
  // Persisted in localStorage so the user stays logged in across
  // browser restarts, not just page refreshes. Tokens expire after
  // 24h server-side (shared/jwt tokenTTL) regardless of how long
  // they sit here, which bounds the risk of this choice.
  const [token, setTokenState] = useState(() => localStorage.getItem(STORAGE_KEY));

  const userID = token ? decodeUserID(token) : null;

  function loginWithToken(newToken) {
    localStorage.setItem(STORAGE_KEY, newToken);
    setTokenState(newToken);
  }

  function logout() {
    localStorage.removeItem(STORAGE_KEY);
    setTokenState(null);
  }

  return (
    <AuthContext.Provider value={{ token, userID, loginWithToken, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
