import { createContext, useContext, useState } from "react";

const AuthContext = createContext(null);

// Decodes the JWT payload client-side purely for the UI's own use
// (knowing the logged-in user's ID) — this is never a security check;
// the server independently re-verifies the token on every request.
function decodeUserID(token) {
  try {
    const payload = token.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(json).user_id;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }) {
  const [token, setToken] = useState(null);

  const userID = token ? decodeUserID(token) : null;

  function loginWithToken(newToken) {
    setToken(newToken);
  }

  function logout() {
    setToken(null);
  }

  return (
    <AuthContext.Provider value={{ token, userID, loginWithToken, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
