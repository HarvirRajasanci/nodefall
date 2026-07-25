import { Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../AuthContext";

export default function HomePage() {
  const { token, userID, logout } = useAuth();
  const navigate = useNavigate();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-900 gap-4">
      <h1 className="text-2xl font-semibold text-gray-100">
        Welcome, {userID?.slice(0, 8)}
      </h1>
      <button
        onClick={() => navigate("/play")}
        className="bg-green-600 hover:bg-green-500 text-white rounded px-6 py-3 font-medium"
      >
        Play Now
      </button>
      <button onClick={logout} className="text-gray-400 hover:underline text-sm">
        Log out
      </button>
    </div>
  );
}
