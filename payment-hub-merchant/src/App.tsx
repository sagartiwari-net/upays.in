import { Navigate, NavLink, Outlet, Route, Routes, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api, clearToken, copyText, getToken, setToken, Merchant, MerchantUser, DashboardStats, Order, SubscriptionUsage } from './api'

function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const res = await api.login(email, password)
      setToken(res.token)
      if (!res.user.onboarding_done) nav('/onboarding')
      else nav('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    }
  }

  return (
    <div className="login-page">
      <div className="login-box">
        <div className="login-logo">
          <div className="logo-icon"><i className="fas fa-bolt" /></div>
          <span className="logo-text">UPIPays</span>
        </div>
        <h1>Merchant Login</h1>
        <p>Sign in to your payment dashboard</p>
        <form onSubmit={submit}>
          <label>Email</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} required />
          <label>Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
          {error && <div className="error">{error}</div>}
          <button className="btn" type="submit" style={{ width: '100%', marginTop: 16, justifyContent: 'center' }}>
            Sign in
          </button>
        </form>
        <p className="muted" style={{ marginTop: 16, textAlign: 'center' }}>
          New here? <NavLink to="/register">Create account</NavLink>
        </p>
      </div>
    </div>
  )
}

function Register() {
  const [form, setForm] = useState({ email: '', password: '', name: '', business_name: '', domain: '' })
  const [error, setError] = useState('')
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const res = await api.register(form)
      setToken(res.token)
      nav('/onboarding')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    }
  }

  return (
    <div className="login-page">
      <div className="login-box" style={{ maxWidth: 480 }}>
        <div className="login-logo">
          <div className="logo-icon"><i className="fas fa-bolt" /></div>
          <span className="logo-text">UPIPays</span>
        </div>
        <h1>Create account</h1>
        <p>Start accepting UPI payments in minutes</p>
        <p className="muted" style={{ marginBottom: 16, textAlign: 'center', fontSize: 14 }}>Free trial: 20 QR codes · 7 days</p>
        <form onSubmit={submit}>
          <label>Your name</label>
          <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
          <label>Business name</label>
          <input value={form.business_name} onChange={e => setForm({ ...form, business_name: e.target.value })} required />
          <label>Website domain</label>
          <input value={form.domain} onChange={e => setForm({ ...form, domain: e.target.value })} placeholder="myshop.com" required />
          <label>Email</label>
          <input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} required />
          <label>Password (min 8 chars)</label>
          <input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} minLength={8} required />
          {error && <div className="error">{error}</div>}
          <button className="btn" type="submit" style={{ width: '100%', marginTop: 16 }}>Create account</button>
        </form>
        <p className="muted" style={{ marginTop: 16, textAlign: 'center' }}>
          Already have account? <NavLink to="/login">Sign in</NavLink>
        </p>
      </div>
    </div>
  )
}

function Onboarding() {
  const nav = useNavigate()
  const [error, setError] = useState('')
  const [parsers, setParsers] = useState<{ id: string; label: string; sender_filter: string }[]>([])
  const [form, setForm] = useState({
    name: 'My UPI',
    upi_id: '',
    payee_name: 'UPIPays',
    imap_host: 'imap.gmail.com',
    imap_port: 993,
    imap_user: '',
    imap_password: '',
    sender_filter: 'hdfcbank',
    parser_type: 'hdfc',
    is_active: true,
  })
  const [webhook, setWebhook] = useState('')

  useEffect(() => {
    api.parserTypes().then(r => {
      setParsers(r.parser_types)
      if (r.parser_types.length) {
        setForm(f => ({ ...f, parser_type: r.parser_types[0].id, sender_filter: r.parser_types[0].sender_filter }))
      }
    }).catch(() => {})
  }, [])

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.setupProfile(form)
      if (webhook.trim()) {
        await api.updateMerchant({ webhook_url: webhook.trim() })
      }
      nav('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Setup failed')
    }
  }

  return (
    <div className="login-page">
      <form className="login-box form" style={{ maxWidth: 520 }} onSubmit={submit}>
        <h1>Setup UPI payments</h1>
        <p className="muted" style={{ marginBottom: 20 }}>Add your UPI ID and Gmail for automatic payment verification</p>
        <label>Profile name</label>
        <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
        <label>UPI ID</label>
        <input value={form.upi_id} onChange={e => setForm({ ...form, upi_id: e.target.value })} placeholder="name@okaxis" required />
        <label>Payee name (shows on QR)</label>
        <input value={form.payee_name} onChange={e => setForm({ ...form, payee_name: e.target.value })} />
        <label>Bank</label>
        <select value={form.parser_type} onChange={e => {
          const pt = parsers.find(p => p.id === e.target.value)
          setForm({ ...form, parser_type: e.target.value, sender_filter: pt?.sender_filter || form.sender_filter })
        }}>
          {(parsers.length ? parsers : [{ id: 'hdfc', label: 'HDFC', sender_filter: 'hdfcbank' }]).map(p => (
            <option key={p.id} value={p.id}>{p.label}</option>
          ))}
        </select>
        <label>Gmail (IMAP)</label>
        <input value={form.imap_user} onChange={e => setForm({ ...form, imap_user: e.target.value })} required />
        <label>Gmail App Password</label>
        <input type="password" value={form.imap_password} onChange={e => setForm({ ...form, imap_password: e.target.value })} required />
        <label>Webhook URL (optional)</label>
        <input value={webhook} onChange={e => setWebhook(e.target.value)} placeholder="https://yoursite.com/webhook" />
        {error && <div className="error">{error}</div>}
        <button className="btn" type="submit" style={{ width: '100%', marginTop: 16 }}>Complete setup</button>
      </form>
    </div>
  )
}

function Shell() {
  const nav = useNavigate()
  const [user, setUser] = useState<MerchantUser | null>(null)

  useEffect(() => {
    api.me().then(r => {
      setUser(r.user)
      if (!r.user.onboarding_done) nav('/onboarding')
    }).catch(() => nav('/login'))
  }, [nav])

  if (!getToken()) return <Navigate to="/login" replace />

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="logo">
          <div className="logo-icon"><i className="fas fa-bolt" /></div>
          <span className="logo-text">UPIPays</span>
        </div>
        <div className="user-profile">
          <div className="user-avatar">{(user?.name || 'M')[0].toUpperCase()}</div>
          <div className="user-info">
            <h3>{user?.name || 'Merchant'}</h3>
            <p>{user?.email}</p>
          </div>
        </div>
        <nav className="nav">
          <NavLink to="/" end><i className="fas fa-chart-line" /> Dashboard</NavLink>
          <NavLink to="/orders"><i className="fas fa-receipt" /> Transactions</NavLink>
          <NavLink to="/billing"><i className="fas fa-credit-card" /> Billing</NavLink>
          <NavLink to="/settings"><i className="fas fa-cog" /> Settings</NavLink>
        </nav>
        <div className="logout">
          <button className="btn secondary" onClick={() => { clearToken(); nav('/login') }}>
            <i className="fas fa-sign-out-alt" /> Sign out
          </button>
        </div>
      </aside>
      <main className="main"><Outlet /></main>
    </div>
  )
}

function UsageWidget({ sub }: { sub: SubscriptionUsage | null }) {
  if (!sub) return null
  const nearLimit = sub.usage_percent >= 80
  const atLimit = sub.orders_used >= sub.order_limit
  return (
    <div className={`card usage-card${atLimit ? ' usage-danger' : nearLimit ? ' usage-warn' : ''}`} style={{ marginBottom: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: 12 }}>
        <div>
          <div className="label">{sub.is_trial ? 'Free trial' : 'Current plan'} — {sub.plan_name}</div>
          <div className="value" style={{ fontSize: 22 }}>
            {sub.orders_used.toLocaleString()} / {sub.order_limit.toLocaleString()} orders
          </div>
          <p className="muted" style={{ marginTop: 6 }}>
            {sub.days_left} days left · expires {new Date(sub.expires_at).toLocaleDateString()}
          </p>
        </div>
        {(atLimit || nearLimit) && (
          <NavLink to="/billing" className="btn" style={{ textDecoration: 'none' }}>
            {atLimit ? 'Upgrade now' : 'View plans'}
          </NavLink>
        )}
      </div>
      <div className="usage-bar" style={{ marginTop: 12 }}>
        <div className="usage-fill" style={{ width: `${Math.min(sub.usage_percent, 100)}%` }} />
      </div>
    </div>
  )
}

function DashboardHome() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [sub, setSub] = useState<SubscriptionUsage | null>(null)
  useEffect(() => {
    api.dashboard().then(setStats).catch(() => {})
    api.subscription().then(r => setSub(r.subscription)).catch(() => {})
  }, [])
  if (!stats) return <p className="muted">Loading…</p>
  return (
    <>
      <header className="page-header"><div><h1>Dashboard</h1><p>Your payment overview</p></div></header>
      <UsageWidget sub={sub} />
      <div className="stats-grid">
        <div className="stat-card green">
          <div className="stat-label">Today&apos;s Revenue</div>
          <div className="stat-value">₹{stats.today_revenue.toFixed(0)}</div>
        </div>
        <div className="stat-card pink">
          <div className="stat-label">Total Transactions</div>
          <div className="stat-value">{stats.total_orders}</div>
        </div>
        <div className="stat-card blue">
          <div className="stat-label">Success Rate</div>
          <div className="stat-value">{stats.success_rate.toFixed(1)}%</div>
        </div>
      </div>
      <div className="cards">
        <div className="card"><div className="label">Today orders</div><div className="value">{stats.today_orders}</div></div>
        <div className="card"><div className="label">Pending</div><div className="value">{stats.pending_orders}</div></div>
        <div className="card"><div className="label">Total revenue</div><div className="value green">₹{stats.total_revenue.toFixed(0)}</div></div>
      </div>
    </>
  )
}

function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([])
  const [total, setTotal] = useState(0)
  useEffect(() => { api.orders().then(r => { setOrders(r.orders); setTotal(r.total) }) }, [])
  return (
    <>
      <header className="page-header"><div><h1>Transactions</h1><p>{total} orders</p></div></header>
      <div className="table-wrap">
        <table>
          <thead><tr><th>Order</th><th>Amount</th><th>Status</th><th>UTR</th><th>Date</th></tr></thead>
          <tbody>
            {orders.map(o => (
              <tr key={o.id}>
                <td><strong>{o.hub_order_id}</strong><br /><span className="muted">{o.merchant_order_id}</span></td>
                <td>₹{o.pay_amount.toFixed(2)}</td>
                <td><span className={`badge ${o.status}`}>{o.status}</span></td>
                <td>{o.customer_utr || '—'}</td>
                <td>{new Date(o.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {orders.length === 0 && <tr><td colSpan={5} className="muted" style={{ textAlign: 'center' }}>No transactions yet</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  )
}

function BillingPage() {
  const [sub, setSub] = useState<SubscriptionUsage | null>(null)
  const [plans, setPlans] = useState<{ id: string; name: string; price_inr: number; order_limit: number; validity_days: number; is_recommended: boolean }[]>([])

  useEffect(() => {
    api.subscription().then(r => setSub(r.subscription)).catch(() => {})
    api.plans().then(r => setPlans(r.plans)).catch(() => {})
  }, [])

  return (
    <>
      <header className="page-header"><div><h1>Billing</h1><p>Subscription & plan limits</p></div></header>
      <UsageWidget sub={sub} />
      <div className="card" style={{ marginBottom: 24 }}>
        <h3 className="card-title">How to upgrade</h3>
        <ol style={{ paddingLeft: 20, lineHeight: 1.8 }}>
          <li>Pay the plan amount via UPI to <strong>upays.in@upi</strong> (or contact support for payment details)</li>
          <li>Email <a href="mailto:support@upays.in">support@upays.in</a> with your registered email and UTR</li>
          <li>We activate your plan within a few hours</li>
        </ol>
      </div>
      <h3 style={{ marginBottom: 16 }}>Available plans</h3>
      <div className="stats-grid">
        {plans.map(p => (
          <div key={p.id} className={`stat-card${p.is_recommended ? ' blue' : ''}`}>
            {p.is_recommended && <span className="badge success" style={{ marginBottom: 8 }}>Recommended</span>}
            <div className="stat-label">{p.name}</div>
            <div className="stat-value">₹{p.price_inr}</div>
            <p className="muted">{p.order_limit.toLocaleString()} orders / {p.validity_days} days</p>
          </div>
        ))}
      </div>
    </>
  )
}

function SettingsPage() {
  const [merchant, setMerchant] = useState<Merchant | null>(null)
  const [secret, setSecret] = useState<{ key: string; secret: string } | null>(null)
  const [form, setForm] = useState({ webhook_url: '', return_url: '' })

  useEffect(() => {
    api.me().then(r => {
      setMerchant(r.merchant)
      setForm({ webhook_url: r.merchant.webhook_url, return_url: r.merchant.return_url })
    })
  }, [])

  async function save(e: React.FormEvent) {
    e.preventDefault()
    const m = await api.updateMerchant(form)
    setMerchant(m)
    alert('Saved')
  }

  async function regen() {
    if (!confirm('Old API secret will stop working. Continue?')) return
    const r = await api.regenerateSecret()
    setSecret({ key: r.api_key, secret: r.api_secret })
  }

  return (
    <>
      <header className="page-header"><div><h1>Settings</h1><p>API keys & webhook</p></div></header>
      {merchant && (
        <div className="card" style={{ marginBottom: 20 }}>
          <h3 className="card-title">API credentials</h3>
          <p className="muted">Hub URL: <code>{merchant.hub_url}</code></p>
          <p>API Key: <code>{merchant.api_key}</code>
            <button type="button" className="btn secondary btn-sm" style={{ marginLeft: 8 }} onClick={() => copyText(merchant.api_key)}>Copy</button>
          </p>
          <button className="btn secondary" onClick={regen}>Regenerate API secret</button>
        </div>
      )}
      {secret && (
        <div className="secret-box" style={{ marginBottom: 20 }}>
          <strong>Save now — secret shown once:</strong>
          <p>Key: <code>{secret.key}</code></p>
          <p>Secret: <code>{secret.secret}</code></p>
          <button className="btn secondary btn-sm" onClick={() => copyText(secret.secret)}>Copy secret</button>
        </div>
      )}
      <form className="form form-wide" onSubmit={save}>
        <label>Webhook URL</label>
        <input value={form.webhook_url} onChange={e => setForm({ ...form, webhook_url: e.target.value })} />
        <label>Return URL</label>
        <input value={form.return_url} onChange={e => setForm({ ...form, return_url: e.target.value })} />
        <button className="btn" type="submit">Save</button>
      </form>
    </>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route path="/onboarding" element={getToken() ? <Onboarding /> : <Navigate to="/login" />} />
      <Route element={<Shell />}>
        <Route index element={<DashboardHome />} />
        <Route path="orders" element={<OrdersPage />} />
        <Route path="billing" element={<BillingPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  )
}
