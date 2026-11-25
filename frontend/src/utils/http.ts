import type { ApiResponse, RequestResult } from '../types/api'

export function resolveApiPrefix(): string {
  if (typeof window !== 'undefined' && window.location.pathname.startsWith('/app/antigravity-harness')) {
    return '/app/antigravity-harness/api'
  }
  return '/api'
}

export interface RequestOptions extends RequestInit {
  timeout?: number
}

class HttpClient {
  private resolveUrl(path: string): string {
    const cleanPath = path.replace(/^\/+/, '')
    return `${resolveApiPrefix()}/${cleanPath}`
  }

  private async request<T>(path: string, options: RequestOptions = {}): Promise<RequestResult<T>> {
    const { timeout = 35000, ...fetchOptions } = options
    const url = this.resolveUrl(path)
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), timeout)

    try {
      const response = await fetch(url, {
        ...fetchOptions,
        signal: controller.signal
      })
      clearTimeout(timer)
      const isJson = response.headers.get('content-type')?.includes('application/json')
      let resBody: ApiResponse<T> | Record<string, unknown> | null = null
      if (isJson) {
        try {
          resBody = await response.json()
        } catch {
          resBody = null
        }
      }
      if (response.ok) {
        if (resBody && typeof resBody === 'object' && 'code' in resBody) {
          const apiResp = resBody as ApiResponse<T>
          if (apiResp.code === 0) {
            return {
              success: true,
              data: apiResp.data,
              message: apiResp.message || 'success',
              timestamp: apiResp.timestamp
            }
          }
          return {
            success: false,
            code: apiResp.code,
            message: apiResp.message || `请求错误 (${apiResp.code})`
          }
        }
        return { success: true, data: resBody as unknown as T, message: 'success' }
      }
      const errMsg = (resBody as { message?: string } | null)?.message || `请求失败 (HTTP ${response.status})`
      return { success: false, code: response.status, message: errMsg }
    } catch (err: unknown) {
      clearTimeout(timer)
      if (err instanceof DOMException && err.name === 'AbortError') {
        return { success: false, code: 408, message: '请求超时，请稍后重试' }
      }
      return { success: false, code: -1, message: (err as Error)?.message || '网络连接异常' }
    }
  }

  get<T>(path: string, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: 'GET' })
  }

  post<T>(path: string, data?: unknown, options?: RequestOptions) {
    return this.request<T>(path, {
      ...options,
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...options?.headers },
      body: data !== undefined ? JSON.stringify(data) : undefined
    })
  }

  patch<T>(path: string, data?: unknown, options?: RequestOptions) {
    return this.request<T>(path, {
      ...options,
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', ...options?.headers },
      body: data !== undefined ? JSON.stringify(data) : undefined
    })
  }

  delete<T>(path: string, options?: RequestOptions) {
    return this.request<T>(path, { ...options, method: 'DELETE' })
  }
}

export const http = new HttpClient()
