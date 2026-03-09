import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const apiKey = localStorage.getItem('api_key') || 'test-api-key-12345'
    config.headers['X-API-Key'] = apiKey
    config.headers['X-Tenant-ID'] = localStorage.getItem('tenant_id') || 'default'
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器 - 直接返回 response.data
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

// 定义 API 类型
export interface Event {
  event_id: string
  trace_id: string
  event_type: string
  timestamp: string
  payload: Record<string, unknown>
  event_hash: string
}

export interface EventDetail extends Event {
  sequence: number
}

export interface Certificate {
  cert_id: string
  trace_id: string
  root_hash: string
  event_count: number
  evidence_level: string
  created_at: string
}

export interface CertificateDetail {
  cert_id: string
  cert_version: string
  schema_version: string
  trace_id: string
  event_hashes: string[]
  root_hash: string
  time_proof?: {
    proof_type: string
    timestamp: string
    signature?: string
  }
  anchor_proof?: {
    anchor_type: string
    anchor_id: string
    anchor_timestamp: string
    storage_provider?: string
    blockchain?: {
      chain_id: string
      tx_hash: string
      block_height: number
    }
  }
  metadata?: {
    tenant_id: string
    created_at: string
    created_by: string
    evidence_level: string
  }
}

export interface VerifyResult {
  valid: boolean
  checks: Record<string, {
    passed: boolean
    message?: string
    timestamp?: string
    type?: string
    anchor_id?: string
  }>
  certificate?: Partial<CertificateDetail> & {
    cert_id?: string
    trace_id?: string
    root_hash?: string
  }
}

// 事件相关API
export const eventApi = {
  search: (params: {
    trace_id?: string
    event_type?: string
    start_time?: string
    end_time?: string
    page?: number
    page_size?: number
  }): Promise<{ events: Event[]; page: number; page_size: number; size: number }> =>
    api.get('/events/search', { params }),

  get: (eventId: string): Promise<EventDetail> =>
    api.get(`/events/${eventId}`),

  ingest: (events: unknown[]): Promise<{ success: boolean; results: unknown[] }> =>
    api.post('/events/ingest', { events }),
}

// 存证相关API
export const certApi = {
  search: (params?: { page?: number; page_size?: number }): Promise<{
    certificates: Certificate[]
    page: number
    page_size: number
    size: number
  }> => api.get('/certs/search', { params }),

  get: (certId: string): Promise<CertificateDetail> =>
    api.get(`/certs/${certId}`),

  commit: (traceId: string, evidenceLevel?: string): Promise<{
    cert_id: string
    trace_id: string
    root_hash: string
    event_count: number
    evidence_level: string
    created_at: string
  }> => api.post('/certs/commit', { trace_id: traceId, evidence_level: evidenceLevel }),

  verify: (certId?: string, rootHash?: string): Promise<VerifyResult> =>
    api.post('/certs/verify', { cert_id: certId, root_hash: rootHash }),

  generateProof: (certId: string, discloseEvents: number[], discloseFields: string[]): Promise<unknown> =>
    api.post(`/certs/${certId}/prove`, {
      disclose_events: discloseEvents,
      disclose_fields: discloseFields,
    }),
}

// 聊天API
export const chatApi = {
  completions: (messages: { role: string; content: string }[], model?: string): Promise<unknown> =>
    api.post('/chat/completions', {
      model: model || 'gpt-3.5-turbo',
      messages,
    }),
}

// 健康检查
export const healthApi = {
  check: (): Promise<{ status: string }> =>
    axios.get('/health').then(res => res.data),
}

export default api
