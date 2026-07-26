const AUTH_URL = "/api/auth";

export async function register(username, password) {
  const res = await fetch(`${AUTH_URL}/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  if (!res.ok) {
    const message = await res.text();
    throw new Error(message || `register failed (${res.status})`);
  }
}

export async function login(username, password) {
  const res = await fetch(`${AUTH_URL}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  if (!res.ok) {
    const message = await res.text();
    throw new Error(message || `login failed (${res.status})`);
  }

  const { token } = await res.json();
  return token;
}
