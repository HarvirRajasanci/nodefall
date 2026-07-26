import { createContext, useContext } from "react";

export const AuthContext = createContext(null);
export const STORAGE_KEY = "nodefall_token";

// Decodes the JWT payload client-side purely for the UI's own use
// (knowing the logged-in user's ID) — this is never a security check;
// the server independently re-verifies the token on every request.
export function decodeUserID(token) {
  try {
    const payload = token.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(json).user_id;
  } catch {
    return null;
  }
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
