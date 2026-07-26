const MATCHMAKER_URL = "/api/matchmaker";

async function authedFetch(path, token, options = {}) {
  const res = await fetch(`${MATCHMAKER_URL}${path}?token=${encodeURIComponent(token)}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options.headers },
  });

  if (!res.ok) {
    const message = await res.text();
    throw new Error(message || `request failed (${res.status})`);
  }

  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

export async function joinQueue(token) {
  return authedFetch("/queue", token, { method: "POST" });
}

export async function queueStatus(token) {
  return authedFetch("/queue", token, { method: "GET" });
}

export async function leaveQueue(token) {
  return authedFetch("/queue", token, { method: "DELETE" });
}
