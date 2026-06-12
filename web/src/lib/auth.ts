const TOKEN_KEY = 'ops_token'
let authExpiredFired = false

export function getAuthToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setAuthToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
  authExpiredFired = false
}

export function clearAuthToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function authHeaders(init?: HeadersInit): HeadersInit {
  const token = getAuthToken()
  const headers: Record<string, string> = {}
  if (init instanceof Headers) {
    init.forEach((v, k) => { headers[k] = v })
  } else if (Array.isArray(init)) {
    init.forEach(([k, v]) => { headers[k] = v })
  } else if (init) {
    Object.assign(headers, init)
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return headers
}

export async function authFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const token = getAuthToken()
  const headers = new Headers(init?.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  if (!headers.has('Content-Type') && init?.body && typeof init.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(input, { ...init, headers })
  if (res.status === 401 && !authExpiredFired) {
    authExpiredFired = true
    clearAuthToken()
    window.dispatchEvent(new Event('auth:expired'))
  }
  return res
}
