import { useState } from "react";
import { useAuthStore } from "../store/authStore";
import { login, register } from "../services/auth";

export default function LoginForm() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isRegisterMode, setIsRegisterMode] = useState(false);
  const { setToken, setLoading, setError, isLoading, error } = useAuthStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const authFn = isRegisterMode ? register : login;
      const response = await authFn({ username, password });
      setToken(response.token, { id: response.user_id, username });

      setTimeout(() => {
        window.location.href = `todoplanner://auth?token=${response.token}`;
      }, 100);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Authentication failed");
    }
  };

  return (
    <div className="w-full max-w-md">
      <div className="bg-navy border border-slate-grey/20 rounded-outer p-8 backdrop-blur">
        <h1 className="text-3xl font-semibold text-snow mb-6 text-center">
          {isRegisterMode ? "Create Account" : "Sign In"}
        </h1>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <label
              htmlFor="username"
              className="block text-sm font-medium text-cloud"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-4 py-3 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro placeholder:text-slate-blue focus:border-cloud focus:bg-navy/80 disabled:opacity-50"
              placeholder="Enter username"
              autoComplete="username"
              autoFocus
              required
            />
          </div>

          <div className="space-y-2">
            <label
              htmlFor="password"
              className="block text-sm font-medium text-cloud"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro placeholder:text-slate-blue focus:border-cloud focus:bg-navy/80 disabled:opacity-50"
              placeholder="Enter password"
              autoComplete={
                isRegisterMode ? "new-password" : "current-password"
              }
              required
            />
          </div>

          {error && (
            <div className="px-4 py-3 bg-error/10 border border-error rounded-lg text-error text-sm text-center">
              {error}
            </div>
          )}

          <button
            type="submit"
            className="w-full h-9 px-6 text-base font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud hover:-translate-y-0.5 active:translate-y-0.5 disabled:opacity-50 disabled:transform-none"
            disabled={isLoading}
          >
            {isLoading
              ? "Processing..."
              : isRegisterMode
                ? "Create Account"
                : "Sign In"}
          </button>
        </form>

        <button
          type="button"
          className="mt-4 w-full text-sm text-cloud underline transition-colors duration-micro hover:text-snow"
          onClick={() => {
            setIsRegisterMode(!isRegisterMode);
            setError(null);
          }}
        >
          {isRegisterMode
            ? "Already have an account? Sign in"
            : "Don't have an account? Create one"}
        </button>
      </div>
    </div>
  );
}
