import { useState, useEffect } from "react";
import { useAuthStore } from "../store/authStore";

export default function TokenDisplay() {
  const { token, user } = useAuthStore();
  const [copied, setCopied] = useState(false);

  // Automatically trigger widget opening when component mounts
  useEffect(() => {
    if (token) {
      const widgetUrl = `todoplanner://auth?token=${token}`;
      console.log("[TOKEN] Attempting to open desktop widget:", widgetUrl);

      // Create a hidden iframe to trigger the URL scheme
      const iframe = document.createElement("iframe");
      iframe.style.display = "none";
      iframe.src = widgetUrl;
      document.body.appendChild(iframe);

      // Clean up after a short delay
      setTimeout(() => {
        document.body.removeChild(iframe);
      }, 1000);
    }
  }, [token]);

  if (!token || !user) return null;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy token:", err);
    }
  };

  return (
    <div className="w-full max-w-2xl">
      <div className="bg-navy border border-slate-grey/20 rounded-outer p-8 backdrop-blur">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 text-3xl text-navy bg-success rounded-full mb-4">
            ✓
          </div>
          <h2 className="text-xl font-semibold text-snow mb-2">
            Authentication Successful
          </h2>
          <p className="text-cloud">Welcome, {user.username}</p>
        </div>

        <div className="space-y-6">
          <p className="text-snow text-center px-4 py-3 bg-success/10 border border-success rounded-lg">
            Your token has been sent to the desktop widget automatically.
          </p>

          <p className="text-sm text-cloud text-center">
            If the desktop widget didn't open, copy your token manually.
          </p>

          <button
            onClick={handleCopy}
            className="w-full h-9 px-6 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud hover:-translate-y-0.5 active:translate-y-0.5"
            aria-label="Copy token to clipboard"
          >
            {copied ? "Copied!" : "Copy to Clipboard"}
          </button>

          <div className="p-4 bg-navy border border-slate-grey/20 rounded-lg">
            <p className="text-sm font-semibold text-cloud mb-2">
              Manual Setup:
            </p>
            <ol className="ml-6 text-cloud text-sm space-y-1 list-decimal">
              <li>Open the desktop widget</li>
              <li>Paste the token into the token field</li>
              <li>Click "Set Token"</li>
            </ol>
          </div>
        </div>
      </div>
    </div>
  );
}
