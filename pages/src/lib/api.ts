const API_BASE = import.meta.env.VITE_API_BASE || "/api";

type HttpMethod = "GET" | "POST";

type ApiResponse<T> = {
  msg: string;
  data: T;
  code: number;
};

async function request<T>(method: HttpMethod, url: string, body?: unknown): Promise<T> {
  const target = url.startsWith("http") ? url : `${API_BASE}${url}`;
  const res = await fetch(target, {
    method,
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  });

  const json = (await res.json().catch(() => null)) as ApiResponse<T> | null;
  if (!json || typeof json.code !== "number") {
    throw new Error("响应格式错误");
  }
  if (json.code !== 200) {
    throw new Error(json.msg || "请求失败");
  }
  return json.data;
}

const api = {
  get: <T>(url: string) => request<T>("GET", url),
  post: <T>(url: string, body?: unknown) => request<T>("POST", url, body),
};

export default api;
