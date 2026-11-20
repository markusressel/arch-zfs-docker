export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// The demo build (`vite build --mode demo`) answers from made-up data in the browser. Vite replaces
// import.meta.env.MODE at build time, so the normal build contains neither the branch nor the demo data.
async function request(path: string, init?: RequestInit): Promise<Response> {
  if (import.meta.env.MODE === 'demo') return (await import('../demo/data')).demoFetch(path, init)
  return fetch(path, init)
}

/** Fetches a path and returns the raw response, throwing ApiError for non-2xx statuses. */
export async function send(path: string, init?: RequestInit): Promise<Response> {
  const res = await request(path, init)
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return res
}

export async function getJson<T>(path: string): Promise<T> {
  return (await send(path)).json() as Promise<T>
}

export function jsonBody(method: string, body: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}
