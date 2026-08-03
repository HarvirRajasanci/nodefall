import { useEffect, useRef, useState } from "react";
import { Navigate, useNavigate, Link } from "react-router-dom";
import { useAuth } from "../authContext";
import { joinQueue, queueStatus, leaveQueue } from "../api/matchmaker";

const POLL_INTERVAL_MS = 1500;

export default function QueuePage() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const joinedRef = useRef(false);

  useEffect(() => {
    if (!token) return;

    let cancelled = false;
    let pollTimer;

    async function poll() {
      try {
        const data = await queueStatus(token);
        if (cancelled) return;

        if (data.status === "matched") {
          navigate(
            `/play?match=${encodeURIComponent(data.match_id)}&server=${encodeURIComponent(data.server_addr)}`
          );
          return;
        }
      } catch (err) {
        if (!cancelled) setError(err.message);
      }

      if (!cancelled) {
        pollTimer = setTimeout(poll, POLL_INTERVAL_MS);
      }
    }

    async function start() {
      try {
        if (!joinedRef.current) {
          await joinQueue(token);
          joinedRef.current = true;
        }
        poll();
      } catch (err) {
        if (!cancelled) setError(err.message);
      }
    }

    start();

    return () => {
      cancelled = true;
      clearTimeout(pollTimer);
      leaveQueue(token).catch(() => {});
    };
  }, [token, navigate]);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6">
      <div className="relative w-14 h-14 flex items-center justify-center">
        <div className="absolute inset-0 border-[1.5px] border-dashed border-emerald-500/60 rounded-full animate-[spin_2s_linear_infinite]" />
        <div className="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center text-gray-950 font-bold text-sm font-mono">
          N
        </div>
      </div>

      <p className="text-gray-400 text-sm font-mono tracking-wide">
        SEARCHING FOR MATCH...
      </p>

      {error && (
        <p className="text-red-400 text-xs bg-red-950/50 border border-red-900 rounded-lg px-3 py-2">
          {error}
        </p>
      )}

      <Link to="/" className="text-gray-500 hover:text-gray-300 text-sm">
        Cancel
      </Link>
    </div>
  );
}
