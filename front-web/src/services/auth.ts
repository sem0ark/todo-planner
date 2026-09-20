import API_URL from "../config";

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user_id: number;
}

export function isTokenExpired(token: string): boolean {
  try {
    const tokenParts = token.split(".");
    if (tokenParts.length !== 3) {
      return true;
    }

    const payloadBase64 = tokenParts[1]
      .replace(/-/g, "+")
      .replace(/_/g, "/");
    const jsonPayload = decodeURIComponent(
      atob(payloadBase64)
        .split("")
        .map((character) => `%${`00${character.charCodeAt(0).toString(16)}`.slice(-2)}`)
        .join(""),
    );

    const payload = JSON.parse(jsonPayload) as { exp?: number };
    if (!payload.exp) {
      return false;
    }

    const currentTimestampSeconds = Math.floor(Date.now() / 1000);
    return currentTimestampSeconds >= payload.exp;
  } catch {
    return true;
  }
}

export async function login(credentials: LoginRequest): Promise<AuthResponse> {
  const response = await fetch(`${API_URL}/auth/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Login failed");
  }

  const data: AuthResponse = await response.json();
  if (!data.token) {
    throw new Error("Invalid response: token is missing");
  }

  if (isTokenExpired(data.token)) {
    throw new Error("Token has already expired");
  }

  return data;
}

export async function register(
  credentials: RegisterRequest,
): Promise<AuthResponse> {
  const response = await fetch(`${API_URL}/auth/register`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Registration failed");
  }

  const data: AuthResponse = await response.json();
  if (!data.token) {
    throw new Error("Invalid response: token is missing");
  }

  if (isTokenExpired(data.token)) {
    throw new Error("Token has already expired");
  }

  return data;
}
