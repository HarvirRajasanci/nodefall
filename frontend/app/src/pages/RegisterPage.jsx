import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { register } from "../api/auth";
import AuthLayout from "../components/AuthLayout";

export default function RegisterPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      await register(username, password);
      setSuccess(true);
      setTimeout(() => navigate("/login"), 1000);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout>
      <h1 className="text-gray-100 text-lg font-medium mb-1">Create account</h1>
      <p className="text-gray-500 text-sm mb-6">Join Nodefall</p>

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
          <p className="text-gray-600 text-xs mt-1.5">At least 8 characters</p>
        </div>

        {error && (
          <p className="text-red-400 text-xs bg-red-950/50 border border-red-900 rounded-lg px-3 py-2">
            {error}
          </p>
        )}
        {success && (
          <p className="text-emerald-400 text-xs bg-emerald-950/50 border border-emerald-900 rounded-lg px-3 py-2">
            Account created — redirecting to login
          </p>
        )}

        <button
          type="submit"
          disabled={loading}
          className="mt-2 bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 disabled:cursor-not-allowed text-gray-950 rounded-lg px-3 py-2.5 text-sm font-medium font-mono tracking-wide transition-colors"
        >
          {loading ? "CREATING ACCOUNT..." : "REGISTER"}
        </button>
      </form>

      <p className="text-gray-500 text-sm text-center mt-6">
        Already have an account?{" "}
        <Link to="/login" className="text-emerald-400 hover:text-emerald-300">
          Log in
        </Link>
      </p>
    </AuthLayout>
  );
}
