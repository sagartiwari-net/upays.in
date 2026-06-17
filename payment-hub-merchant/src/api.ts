const TOKEN_KEY = 'upipays_merchant_token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

const API = '/merchant/api'

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(opts.headers as Record<string, string> || {}),
  }
  const token = getToken()
  if (token) headers.Authorization = 'Bearer ' + token

  const res = await fetch(API + path, { ...opts, headers })
  const data = await res.json().catch(() => ({}))
  if (res.status === 401 && !path.includes('/auth/login') && !path.includes('/auth/register')) {
    clearToken()
    if (!window.location.pathname.endsWith('/login')) {
      window.location.href = '/dashboard/login'
    }
    throw new Error('Session expired')
  }
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || 'Request failed')
  }
  return data as T
}

export type MerchantUser = {
  id: string
  email: string
  name: string
  merchant_id: string
  onboarding_done: boolean
}

export type Merchant = {
  id: string
  name: string
  domain: string
  api_key: string
  api_secret?: string
  webhook_url: string
  return_url: string
  payment_profile_id: string
  hub_url: string
}

export type DashboardStats = {
  today_orders: number
  today_success: number
  today_revenue: number
  total_orders: number
  total_revenue: number
  pending_orders: number
  success_rate: number
}

export type SubscriptionUsage = {
  plan_id: string
  plan_name: string
  plan_slug: string
  plan_price_inr: number
  status: string
  orders_used: number
  order_limit: number
  starts_at: string
  expires_at: string
  days_left: number
  usage_percent: number
  is_trial: boolean
}

export type SubscriptionPlan = {
  id: string
  slug: string
  name: string
  price_inr: number
  validity_days: number
  order_limit: number
  is_recommended: boolean
  features_json: string
}

export type Order = {
  id: string
  hub_order_id: string
  merchant_order_id: string
  amount: number
  pay_amount: number
  status: string
  customer_email: string
  product_name: string
  customer_utr: string
  created_at: string
}

export const api = {
  register: (body: object) => request<{ token: string; user: MerchantUser; merchant: Merchant }>('/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  login: (email: string, password: string) => request<{ token: string; user: MerchantUser; merchant: Merchant }>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  me: () => request<{ user: MerchantUser; merchant: Merchant; payment_profile?: object }>('/auth/me'),
  dashboard: () => request<DashboardStats>('/dashboard'),
  subscription: () => request<{ subscription: SubscriptionUsage | null }>('/subscription'),
  plans: () => request<{ plans: SubscriptionPlan[] }>('/plans'),
  orders: (params?: Record<string, string>) => {
    const q = new URLSearchParams(params).toString()
    return request<{ orders: Order[]; total: number }>('/orders' + (q ? '?' + q : ''))
  },
  updateMerchant: (body: object) => request<Merchant>('/merchant', { method: 'PUT', body: JSON.stringify(body) }),
  regenerateSecret: () => request<{ api_key: string; api_secret: string }>('/merchant/regenerate-secret', { method: 'POST' }),
  setupProfile: (body: object) => request<{ payment_profile: object; onboarding_done: boolean }>('/payment-profile', { method: 'POST', body: JSON.stringify(body) }),
  parserTypes: () => request<{ parser_types: { id: string; label: string; sender_filter: string }[] }>('/parser-types'),
  createPaymentLink: (body: { amount: number; product_name: string; return_url?: string }) =>
    request<{ order_id: string; payment_url: string; expires_at: string }>('/payment-links', { method: 'POST', body: JSON.stringify(body) }),
  downloadWooCommerce: () => download('/downloads/woocommerce-plugin', 'upipays-woocommerce.zip'),
  downloadAmember: () => download('/downloads/amember-plugin', 'upipays-amember-plugin.zip'),
}

async function download(path: string, filename: string) {
  const token = getToken()
  const res = await fetch('/merchant/api' + path, { headers: token ? { Authorization: 'Bearer ' + token } : {} })
  if (!res.ok) throw new Error('Download failed')
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export function copyText(text: string) {
  return navigator.clipboard.writeText(text)
}
