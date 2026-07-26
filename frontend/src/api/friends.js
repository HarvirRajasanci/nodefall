const AUTH_URL = "/api/auth";

async function authedFetch(path, token, options = {}) {
  const res = await fetch(`${AUTH_URL}${path}?token=${encodeURIComponent(token)}`, {
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

export async function listFriends(token) {
  return authedFetch("/friends", token);
}

export async function sendFriendRequest(token, username) {
  return authedFetch("/friends/request", token, {
    method: "POST",
    body: JSON.stringify({ username }),
  });
}

export async function acceptFriendRequest(token, friendshipId) {
  return authedFetch("/friends/accept", token, {
    method: "POST",
    body: JSON.stringify({ friendship_id: friendshipId }),
  });
}

export async function removeFriend(token, friendshipId) {
  return authedFetch(`/friends/${friendshipId}`, token, {
    method: "DELETE",
  });
}
