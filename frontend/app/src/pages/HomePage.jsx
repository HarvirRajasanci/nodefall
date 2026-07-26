import { Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../AuthContext";

export default function HomePage() {
  const { token, userID, logout } = useAuth();
  const navigate = useNavigate();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-8">
      <div className="relative w-16 h-16 flex items-center justify-center">
        <div className="absolute inset-0 border-[1.5px] border-dashed border-emerald-500/60 rounded-full animate-[spin_8s_linear_infinite]" />
        <div className="w-10 h-10 rounded-lg bg-emerald-500 flex items-center justify-center text-gray-950 font-bold font-mono">
          N
        </div>
      </div>

      <div className="text-center">
        <div className="text-gray-100 text-2xl font-medium font-mono tracking-[0.15em]">
          NODEFALL
        </div>
        <div className="text-gray-600 text-xs tracking-[0.08em] mt-1">
          LAST NODE STANDING
        </div>
      </div>

      <p className="text-gray-500 text-sm">
        Signed in as <span className="text-gray-300 font-mono">{userID?.slice(0, 8)}</span>
      </p>

      <div className="flex flex-col gap-3 w-64">
        <button
          onClick={() => navigate("/play")}
          className="bg-emerald-500 hover:bg-emerald-400 text-gray-950 rounded-lg px-4 py-3 text-sm font-medium font-mono tracking-wide transition-colors"
        >
          PLAY NOW
        </button>
        <button
          onClick={logout}
          className="text-gray-500 hover:text-gray-300 text-sm py-2 transition-colors"
        >
          Log out
        </button>
      </div>
    </div>
  );
}
