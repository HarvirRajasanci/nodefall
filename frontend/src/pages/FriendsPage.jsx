import { useEffect, useState } from "react";
import { Link, Navigate } from "react-router-dom";
import { useAuth } from "../AuthContext";
import { listFriends, sendFriendRequest, acceptFriendRequest, removeFriend } from "../api/friends";

export default function FriendsPage() {
  const { token } = useAuth();
  const [friends, setFriends] = useState([]);
  const [pending, setPending] = useState([]);
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(true);

  async function refresh() {
    try {
      const data = await listFriends(token);
      setFriends(data.friends ?? []);
      setPending(data.pending ?? []);
    } catch (err) {
      setError(err.message);
    }
  }

  useEffect(() => {
    if (!token) return;

    let ignore = false;

    async function load() {
      try {
        const data = await listFriends(token);
        if (ignore) return;
        setFriends(data.friends ?? []);
        setPending(data.pending ?? []);
      } catch (err) {
        if (!ignore) setError(err.message);
      } finally {
        if (!ignore) setLoading(false);
      }
    }

    load();
    return () => {
      ignore = true;
    };
  }, [token]);

  async function handleAddFriend(e) {
    e.preventDefault();
    setError("");
    setSuccess("");

    try {
      await sendFriendRequest(token, username);
      setSuccess(`Friend request sent to ${username}`);
      setUsername("");
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleAccept(friendshipId) {
    setError("");
    try {
      await acceptFriendRequest(token, friendshipId);
      await refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleRemove(friendshipId) {
    setError("");
    try {
      await removeFriend(token, friendshipId);
      await refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen flex flex-col items-center py-16 gap-8">
      <div className="w-full max-w-md flex items-center justify-between">
        <h1 className="text-gray-100 text-xl font-medium font-mono tracking-wide">FRIENDS</h1>
        <Link to="/" className="text-gray-500 hover:text-gray-300 text-sm">
          ← Home
        </Link>
      </div>

      <form
        onSubmit={handleAddFriend}
        className="relative bg-gray-900 border border-gray-800 rounded-xl p-6 w-full max-w-md flex gap-2"
      >
        <div className="absolute -top-px -left-px w-4 h-4 border-t-2 border-l-2 border-emerald-500 rounded-tl-xl" />
        <div className="absolute -bottom-px -right-px w-4 h-4 border-b-2 border-r-2 border-emerald-500 rounded-br-xl" />
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          className="flex-1 bg-gray-800 text-gray-100 rounded-lg px-3 py-2 text-sm border border-gray-700 outline-none focus:border-emerald-500 transition-colors"
        />
        <button
          type="submit"
          className="bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg px-4 py-2 text-sm font-medium font-mono tracking-wide transition-colors"
        >
          ADD
        </button>
      </form>

      {error && (
        <p className="text-red-400 text-xs bg-red-950/50 border border-red-900 rounded-lg px-3 py-2 w-full max-w-md">
          {error}
        </p>
      )}
      {success && (
        <p className="text-emerald-400 text-xs bg-emerald-950/50 border border-emerald-900 rounded-lg px-3 py-2 w-full max-w-md">
          {success}
        </p>
      )}

      {loading ? (
        <p className="text-gray-500 text-sm">Loading...</p>
      ) : (
        <div className="w-full max-w-md flex flex-col gap-6">
          {pending.length > 0 && (
            <div>
              <h2 className="text-gray-400 text-xs tracking-wide mb-2">PENDING REQUESTS</h2>
              <div className="flex flex-col gap-2">
                {pending.map((p) => (
                  <div
                    key={p.friendship_id}
                    className="flex items-center justify-between bg-gray-900 border border-gray-800 rounded-lg px-4 py-2"
                  >
                    <span className="text-gray-100 text-sm">{p.username}</span>
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleAccept(p.friendship_id)}
                        className="text-emerald-400 hover:text-emerald-300 text-xs font-mono tracking-wide"
                      >
                        ACCEPT
                      </button>
                      <button
                        onClick={() => handleRemove(p.friendship_id)}
                        className="text-gray-500 hover:text-gray-300 text-xs font-mono tracking-wide"
                      >
                        DECLINE
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div>
            <h2 className="text-gray-400 text-xs tracking-wide mb-2">
              FRIENDS {friends.length > 0 && `(${friends.length})`}
            </h2>
            {friends.length === 0 ? (
              <p className="text-gray-600 text-sm">No friends yet — add one above.</p>
            ) : (
              <div className="flex flex-col gap-2">
                {friends.map((f) => (
                  <div
                    key={f.friendship_id}
                    className="flex items-center justify-between bg-gray-900 border border-gray-800 rounded-lg px-4 py-2"
                  >
                    <span className="text-gray-100 text-sm">{f.username}</span>
                    <button
                      onClick={() => handleRemove(f.friendship_id)}
                      className="text-gray-500 hover:text-red-400 text-xs font-mono tracking-wide"
                    >
                      REMOVE
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
