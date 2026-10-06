export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

/** Fetches a path and returns the raw response, throwing ApiError for non-2xx statuses. */
export async function send(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(path, init)
  if (!res.ok) throw new ApiError(res.status, (await res.text()).trim() || res.statusText)
  return res
}

export async function getJson<T>(path: string): Promise<T> {
  return (await send(path)).json() as Promise<T>
}

export function jsonBody(method: string, body: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}
