const TOKEN_KEY = 'upipays_admin_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

const API = import.meta.env.BASE_URL + 'api'
const LOGIN_PATH = import.meta.env.BASE_URL + 'login'

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(opts.headers as Record<string, string> || {}),
  }
  const token = getToken()
  if (token) headers.Authorization = 'Bearer ' + token

  let res: Response
  try {
    res = await fetch(API + path, { ...opts, headers })
  } catch {
    throw new Error('Network error — could not reach Payment Hub API')
  }

  const data = await res.json().catch(() => ({}))
  const isLogin = path.includes('/auth/login')
  if (res.status === 401 && !isLogin) {
    clearToken()
    if (!window.location.pathname.endsWith('/login')) {
      window.location.href = LOGIN_PATH
    }
    throw new Error('Session expired — please sign in again')
  }
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`)
  }
  return data as T
}

async function download(path: string, filename: string) {
  const token = getToken()
  const res = await fetch(API + path, {
    headers: token ? { Authorization: 'Bearer ' + token } : {},
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error(data.error || 'Download failed')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export const api = {
  login: (email: string, password: string) =>
    request<{ token: string; admin: { email: string; name: string } }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  dashboard: () => request<DashboardStats>('/dashboard'),
  merchantRevenue: (days = 30) =>
    request<{ merchants: MerchantRevenue[]; days: number }>('/dashboard/merchant-revenue?days=' + days),
  imapAlerts: () => request<{ alerts: IMAPAlert[] }>('/dashboard/imap-alerts'),
  orders: (params: Record<string, string>) => {
    const q = new URLSearchParams(params).toString()
    return request<{ orders: Order[]; total: number }>('/orders?' + q)
  },
  exportOrders: (params: Record<string, string>) => {
    const q = new URLSearchParams(params).toString()
    return download('/orders/export?' + q, 'transactions.csv')
  },
  manualApprove: (orderId: string, body: { utr: string; amount?: number; bank_txn_id?: string }) =>
    request<{ ok: boolean; hub_order_id: string }>('/orders/' + orderId + '/manual-approve', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  merchants: () => request<{ merchants: Merchant[] }>('/merchants'),
  getMerchant: (id: string) => request<Merchant>('/merchants/' + id),
  createMerchant: (body: Partial<Merchant>) =>
    request<Merchant>('/merchants', { method: 'POST', body: JSON.stringify(body) }),
  updateMerchant: (id: string, body: Partial<Merchant>) =>
    request<Merchant>('/merchants/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  assignProfile: (merchantId: string, payment_profile_id: string) =>
    request<{ ok: boolean }>('/merchants/' + merchantId + '/payment-profile', {
      method: 'PUT',
      body: JSON.stringify({ payment_profile_id }),
    }),
  regenerateSecret: (id: string) =>
    request<{ api_key: string; api_secret: string; message: string }>(
      '/merchants/' + id + '/regenerate-secret',
      { method: 'POST' },
    ),
  onboardWebsite: (body: OnboardWebsiteInput) =>
    request<OnboardWebsiteResult>('/onboarding/website', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  parserTypes: () =>
    request<{ parser_types: ParserType[] }>('/payment-profiles/parser-types'),
  profiles: () => request<{ profiles: Profile[] }>('/payment-profiles'),
  createProfile: (body: Partial<Profile>) =>
    request<Profile>('/payment-profiles', { method: 'POST', body: JSON.stringify(body) }),
  updateProfile: (id: string, body: Partial<Profile>) =>
    request<Profile>('/payment-profiles/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  testIMAP: (id: string) =>
    request<{ ok: boolean; message: string; subjects: string[] }>(
      '/payment-profiles/' + id + '/test-imap',
      { method: 'POST' },
    ),
  testParse: (id: string, body: { email_body?: string; fetch_latest?: boolean }) =>
    request<{ matched: boolean; amount?: number; utr?: string; message?: string }>(
      '/payment-profiles/' + id + '/test-parse',
      { method: 'POST', body: JSON.stringify(body) },
    ),
  triggerPoll: (id: string) =>
    request<{ ok: boolean; message: string }>('/payment-profiles/' + id + '/trigger-poll', {
      method: 'POST',
    }),
  webhooks: (offset = '0') =>
    request<{ webhooks: Webhook[]; total: number }>('/webhooks?offset=' + offset),
  unmatched: (offset = '0') =>
    request<{ unmatched: UnmatchedTxn[]; total: number }>('/unmatched?offset=' + offset),
  downloadPlugin: () => download('/downloads/amember-plugin', 'upipays-amember-plugin.zip'),
}

export type DashboardStats = {
  today_orders: number
  today_success: number
  today_revenue: number
  total_orders: number
  total_success: number
  total_revenue: number
  pending_orders: number
  success_rate: number
}

export type MerchantRevenue = {
  merchant_id: string
  merchant_name: string
  domain: string
  orders: number
  revenue: number
}

export type IMAPAlert = {
  id: string
  name: string
  imap_user: string
  imap_last_ok_at?: string
  imap_last_error?: string
  imap_last_checked_at?: string
}

export type Order = {
  id: string
  hub_order_id: string
  merchant_order_id: string
  merchant_name: string
  merchant_domain: string
  amount: number
  pay_amount: number
  status: string
  customer_email: string
  product_name: string
  customer_utr: string
  paid_at?: string
  created_at: string
}

export type Merchant = {
  id: string
  name: string
  domain: string
  api_key: string
  api_secret: string
  webhook_url: string
  return_url: string
  status: string
  payment_profile_id: string
}

export type ParserType = {
  id: string
  label: string
  sender_filter: string
  bank_code: string
}

export type Profile = {
  id: string
  name: string
  upi_id: string
  payee_name: string
  imap_host: string
  imap_port: number
  imap_user: string
  imap_password: string
  sender_filter: string
  parser_type: string
  is_active: boolean
  imap_last_ok_at?: string
  imap_last_error?: string
  imap_last_checked_at?: string
}

export type Webhook = {
  id: string
  hub_order_id: string
  merchant_name: string
  direction: string
  status: string
  response_code?: number
  created_at: string
}

export type UnmatchedTxn = {
  id: string
  utr: string
  amount: number
  profile_id: string
  profile_name: string
  raw_excerpt: string
  created_at: string
}

export type OnboardWebsiteInput = {
  merchant: {
    name: string
    domain: string
    webhook_url: string
    return_url?: string
  }
  payment: {
    mode: 'existing' | 'new'
    profile_id?: string
    profile?: Partial<Profile>
  }
}

export type OnboardWebsiteResult = {
  merchant: Merchant
  payment_profile?: Profile
  checklist: {
    amember_plugin_path: string
    hub_url: string
    webhook_url: string
  }
}

export function copyText(text: string) {
  return navigator.clipboard.writeText(text)
}

export function imapHealthDot(p: Profile): 'ok' | 'warn' | 'fail' | 'unknown' {
  if (p.imap_last_error && !p.imap_last_ok_at) return 'fail'
  if (p.imap_last_error) return 'warn'
  if (p.imap_last_ok_at) {
    const ok = new Date(p.imap_last_ok_at).getTime()
    if (Date.now() - ok > 3600000) return 'warn'
    return 'ok'
  }
  return 'unknown'
}
