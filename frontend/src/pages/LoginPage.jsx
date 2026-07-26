import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { login } from "../api/auth";
import { useAuth } from "../authContext";
import AuthLayout from "../components/AuthLayout";

export default function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const { loginWithToken } = useAuth();
  const navigate = useNavigate();

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const token = await login(username, password);
      loginWithToken(token);
      navigate("/");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout>
      <h1 className="text-gray-100 text-lg font-medium mb-1">Welcome back</h1>
      <p className="text-gray-500 text-sm mb-6">Log in to play</p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <div>
          <label className="block text-gray-400 text-xs mb-1.5">Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full bg-gray-800 text-gray-100 rounded-lg px-3 py-2.5 text-sm border border-gray-700 outline-none focus:border-emerald-500 transition-colors"
          />
        </div>

        <div>
          <label className="block text-gray-400 text-xs mb-1.5">Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full bg-gray-800 text-gray-100 rounded-lg px-3 py-2.5 text-sm border border-gray-700 outline-none focus:border-emerald-500 transition-colors"
          />
        </div>

        {error && (
          <p className="text-red-400 text-xs bg-red-950/50 border border-red-900 rounded-lg px-3 py-2">
            {error}
          </p>
        )}

        <button
          type="submit"
          disabled={loading}
          className="mt-2 bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 disabled:cursor-not-allowed text-gray-950 rounded-lg px-3 py-2.5 text-sm font-medium font-mono tracking-wide transition-colors"
        >
          {loading ? "LOGGING IN..." : "LOG IN"}
        </button>
      </form>

      <p className="text-gray-500 text-sm text-center mt-6">
        No account?{" "}
        <Link to="/register" className="text-emerald-400 hover:text-emerald-300">
          Register
        </Link>
      </p>
    </AuthLayout>
  );
}
