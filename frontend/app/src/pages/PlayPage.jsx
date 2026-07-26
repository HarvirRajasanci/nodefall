import { useEffect, useRef, useState } from "react";
import { Navigate, Link } from "react-router-dom";
import { useAuth } from "../AuthContext";

const GAME_WS_URL = "ws://localhost:8081/ws";

const PLAYER_RADIUS = 16;
const BULLET_RADIUS = 6;
const ITEM_RADIUS = 18;

export default function PlayPage() {
  const { token, userID } = useAuth();
  const canvasRef = useRef(null);
  const hudRef = useRef(null);

  const [connectionPhase, setConnectionPhase] = useState("connecting");
  const [matchPhase, setMatchPhase] = useState("waiting");
  const [countdown, setCountdown] = useState(0);

  useEffect(() => {
    if (!token) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    let state = { players: [], bullets: [], items: [], zone: null };
    let camera = { x: 0, y: 0 };
    const input = { dx: 0, dy: 0, angle: 0, shoot: false };
    const keys = {};

    const ws = new WebSocket(`${GAME_WS_URL}?token=${encodeURIComponent(token)}`);

    ws.onopen = () => {
      setHud(`connected as ${userID?.slice(0, 8)}`);
      setConnectionPhase("connected");
    };
    ws.onclose = () => setHud("disconnected");
    ws.onerror = () => setHud("connection error");
    ws.onmessage = (event) => {
      state = JSON.parse(event.data);
      setMatchPhase(state.phase);
      setCountdown(state.countdown_seconds ?? 0);
    };

    function setHud(text) {
      if (hudRef.current) hudRef.current.textContent = text;
    }

    function handleKeyDown(e) {
      keys[e.key.toLowerCase()] = true;
    }
    function handleKeyUp(e) {
      keys[e.key.toLowerCase()] = false;
    }
    function handleMouseMove(e) {
      const dx = e.clientX - canvas.width / 2;
      const dy = e.clientY - canvas.height / 2;
      input.angle = Math.atan2(dy, dx);
    }
    function handleMouseDown() {
      input.shoot = true;
    }
    function handleMouseUp() {
      input.shoot = false;
    }
    function handleResize() {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    }

    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    canvas.addEventListener("mousemove", handleMouseMove);
    canvas.addEventListener("mousedown", handleMouseDown);
    canvas.addEventListener("mouseup", handleMouseUp);
    window.addEventListener("resize", handleResize);

    const inputInterval = setInterval(() => {
      if (ws.readyState !== WebSocket.OPEN) return;

      input.dx = (keys["d"] ? 1 : 0) - (keys["a"] ? 1 : 0);
      input.dy = (keys["s"] ? 1 : 0) - (keys["w"] ? 1 : 0);

      ws.send(JSON.stringify(input));
    }, 50);

    function worldToScreen(x, y) {
      return {
        x: x - camera.x + canvas.width / 2,
        y: y - camera.y + canvas.height / 2,
      };
    }

    let animationFrameId;

    function draw() {
      ctx.fillStyle = "#111";
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      const self = state.players?.find((p) => p.id === userID);
      if (self) {
        camera.x = self.x;
        camera.y = self.y;
      }

      if (state.zone) {
        const c = worldToScreen(state.zone.x, state.zone.y);
        ctx.strokeStyle = "#4af";
        ctx.lineWidth = 3;
        ctx.beginPath();
        ctx.arc(c.x, c.y, state.zone.radius, 0, Math.PI * 2);
        ctx.stroke();
      }

      ctx.fillStyle = "#fc4";
      for (const item of state.items ?? []) {
        const p = worldToScreen(item.x, item.y);
        ctx.beginPath();
        ctx.arc(p.x, p.y, ITEM_RADIUS, 0, Math.PI * 2);
        ctx.fill();
      }

      ctx.fillStyle = "#ff4";
      for (const bullet of state.bullets ?? []) {
        const p = worldToScreen(bullet.x, bullet.y);
        ctx.beginPath();
        ctx.arc(p.x, p.y, BULLET_RADIUS, 0, Math.PI * 2);
        ctx.fill();
      }

      for (const player of state.players ?? []) {
        if (!player.alive) continue;
        const p = worldToScreen(player.x, player.y);

        ctx.fillStyle = player.id === userID ? "#4f8" : "#f44";
        ctx.beginPath();
        ctx.arc(p.x, p.y, PLAYER_RADIUS, 0, Math.PI * 2);
        ctx.fill();

        ctx.strokeStyle = "#fff";
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(p.x, p.y);
        ctx.lineTo(
          p.x + Math.cos(player.angle) * PLAYER_RADIUS * 1.5,
          p.y + Math.sin(player.angle) * PLAYER_RADIUS * 1.5
        );
        ctx.stroke();

        ctx.fillStyle = "#333";
        ctx.fillRect(p.x - 20, p.y - PLAYER_RADIUS - 12, 40, 5);
        ctx.fillStyle = "#4f8";
        ctx.fillRect(p.x - 20, p.y - PLAYER_RADIUS - 12, 40 * (player.hp / 100), 5);
      }

      const self2 = state.players?.find((p) => p.id === userID);
      if (self2) {
        setHud(
          `${userID.slice(0, 8)} — HP ${self2.hp}  Armour ${self2.armour}  Gun ${self2.gun}${
            self2.alive ? "" : "  (DEAD)"
          }`
        );
      }

      animationFrameId = requestAnimationFrame(draw);
    }

    animationFrameId = requestAnimationFrame(draw);

    return () => {
      cancelAnimationFrame(animationFrameId);
      clearInterval(inputInterval);
      ws.close();
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      canvas.removeEventListener("mousemove", handleMouseMove);
      canvas.removeEventListener("mousedown", handleMouseDown);
      canvas.removeEventListener("mouseup", handleMouseUp);
      window.removeEventListener("resize", handleResize);
    };
  }, [token, userID]);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  const showOverlay = connectionPhase === "connecting" || matchPhase === "waiting";

  return (
    <div className="relative">
      <div
        ref={hudRef}
        className={`fixed top-2 left-2 text-gray-100 font-mono text-sm z-10 transition-opacity ${
          showOverlay ? "opacity-0" : "opacity-100"
        }`}
      >
        Not connected
      </div>
      <Link
        to="/"
        className="fixed top-2 right-2 text-gray-100 font-mono text-sm z-10 hover:underline"
      >
        ← Home
      </Link>

      {connectionPhase === "connecting" && (
        <div className="fixed inset-0 z-20 flex flex-col items-center justify-center gap-4 bg-gray-950">
          <div className="relative w-14 h-14 flex items-center justify-center">
            <div className="absolute inset-0 border-[1.5px] border-dashed border-emerald-500/60 rounded-full animate-[spin_1.2s_linear_infinite]" />
            <div className="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center text-gray-950 font-bold text-sm font-mono">
              N
            </div>
          </div>
          <p className="text-gray-400 text-sm font-mono tracking-wide">
            CONNECTING TO GAME SERVER...
          </p>
        </div>
      )}

      {connectionPhase === "connected" && matchPhase === "waiting" && (
        <div className="fixed inset-0 z-20 flex flex-col items-center justify-center gap-3 bg-gray-950/90">
          <div className="text-emerald-400 text-7xl font-mono font-bold tabular-nums">
            {countdown > 0 ? countdown : "GO"}
          </div>
          <p className="text-gray-500 text-sm font-mono tracking-wide">
            MATCH STARTING
          </p>
        </div>
      )}

      <canvas ref={canvasRef} className="block bg-gray-900" />
    </div>
  );
}
