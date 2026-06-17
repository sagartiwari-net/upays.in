import { Navigate, NavLink, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api, clearToken, getToken, setToken, copyText, imapHealthDot, DashboardStats, MerchantRevenue, IMAPAlert, Order, Merchant, Profile, ParserType, SubscriptionPlan, SubscriptionUsage, CMSPage } from './api'
import AddWebsite from './AddWebsite'
import Unmatched from './Unmatched'
import Webhooks from './Webhooks'

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
      nav('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    }
  }

  return (
    <div className="login-page">
      <div className="login-box">
        <div className="login-logo">
          <div className="logo-icon"><i className="fas fa-rocket" /></div>
          <span className="logo-text">UPIPays</span>
        </div>
        <h1>Payment Hub</h1>
        <p>Sign in to manage merchants, UPI profiles & transactions</p>
        <form onSubmit={submit}>
          <label>Email</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} required />
          <label>Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
          {error && <div className="error">{error}</div>}
          <div className="actions" style={{ marginTop: 16 }}>
            <button className="btn" type="submit" style={{ width: '100%', justifyContent: 'center' }}>
              <i className="fas fa-sign-in-alt" /> Sign in
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

const PAGE_META: Record<string, { title: string; subtitle: string }> = {
  '/': { title: 'Dashboard', subtitle: "Welcome back! Here's your payment overview" },
  '/orders': { title: 'Transactions', subtitle: 'View and manage all UPI transactions' },
  '/unmatched': { title: 'Unmatched', subtitle: 'Bank payments that could not be auto-matched' },
  '/webhooks': { title: 'Webhooks', subtitle: 'Delivery logs for merchant webhook callbacks' },
  '/websites/add': { title: 'Add Website', subtitle: 'Onboard a new merchant site' },
  '/merchants': { title: 'Manage Keys', subtitle: 'API keys and merchant credentials' },
  '/profiles': { title: 'UPI Profiles', subtitle: 'Configure UPI + IMAP for auto-verification' },
  '/plans': { title: 'Pricing Plans', subtitle: 'Edit subscription plans shown on marketing site' },
  '/pages': { title: 'CMS Pages', subtitle: 'Create dynamic pages like /about without deploy' },
}

function PageHeader() {
  const { pathname } = useLocation()
  const meta = PAGE_META[pathname] || { title: 'Payment Hub', subtitle: '' }
  return (
    <header className="page-header">
      <div>
        <h1>{meta.title}</h1>
        {meta.subtitle && <p>{meta.subtitle}</p>}
      </div>
    </header>
  )
}

function Shell() {
  const nav = useNavigate()
  if (!getToken()) return <Navigate to="/login" replace />

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="logo">
          <div className="logo-icon"><i className="fas fa-rocket" /></div>
          <span className="logo-text">UPIPays</span>
        </div>
        <div className="user-profile">
          <div className="user-avatar">A</div>
          <div className="user-info">
            <h3>Admin</h3>
            <p>Payment Gateway</p>
          </div>
        </div>
        <nav className="nav">
          <NavLink to="/" end><i className="fas fa-chart-line" /> Dashboard</NavLink>
          <NavLink to="/orders"><i className="fas fa-exchange-alt" /> Transactions</NavLink>
          <NavLink to="/merchants"><i className="fas fa-key" /> Manage Keys</NavLink>
          <NavLink to="/profiles"><i className="fas fa-wallet" /> UPI Profiles</NavLink>
          <div className="nav-divider" />
          <NavLink to="/plans"><i className="fas fa-tags" /> Plans</NavLink>
          <NavLink to="/pages"><i className="fas fa-file-alt" /> Pages</NavLink>
          <div className="nav-divider" />
          <NavLink to="/websites/add"><i className="fas fa-plus-circle" /> Add Website</NavLink>
          <NavLink to="/unmatched"><i className="fas fa-unlink" /> Unmatched</NavLink>
          <NavLink to="/webhooks"><i className="fas fa-bell" /> Webhooks</NavLink>
        </nav>
        <div className="logout">
          <button className="btn secondary" onClick={() => { clearToken(); nav('/login') }}>
            <i className="fas fa-sign-out-alt" /> Sign out
          </button>
        </div>
      </aside>
      <main className="main">
        <PageHeader />
        <Outlet />
      </main>
    </div>
  )
}

function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [revenue, setRevenue] = useState<MerchantRevenue[]>([])
  const [alerts, setAlerts] = useState<IMAPAlert[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    setError('')
    api.dashboard()
      .then(setStats)
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load dashboard'))
    api.merchantRevenue(30).then(r => setRevenue(r.merchants)).catch(() => {})
    api.imapAlerts().then(r => setAlerts(r.alerts)).catch(() => {})
  }, [])

  if (error) {
    return (
      <div className="alert-banner" style={{ borderColor: '#ef4444', background: 'rgba(239,68,68,0.08)' }}>
        <strong>Could not load data:</strong> {error}
        <p className="muted" style={{ marginTop: 8 }}>
          Try logout → login again. If new deploy, run migration 000008 on VPS database.
        </p>
        <button className="btn secondary btn-sm" style={{ marginTop: 12 }} onClick={() => window.location.reload()}>Retry</button>
      </div>
    )
  }

  if (!stats) return <p className="muted">Loading…</p>
  const maxRev = Math.max(...revenue.map(r => r.revenue), 1)

  return (
    <>
      {alerts.length > 0 && (
        <div className="alert-banner">
          <strong><i className="fas fa-exclamation-triangle" /> IMAP alert:</strong> {alerts.length} profile(s) — email poll 1+ hour stale.
          <ul style={{ marginTop: 8, paddingLeft: 20 }}>
            {alerts.map(a => (
              <li key={a.id}>{a.name} ({a.imap_user}) — {a.imap_last_error || 'last OK > 1h ago'}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="dashboard-grid">
        <div className="card revenue-card">
          <h3 className="card-title">Total Revenue</h3>
          <div className="revenue-amount">₹{stats.total_revenue.toLocaleString('en-IN', { maximumFractionDigits: 0 })}</div>
          <div className="revenue-meta">
            <i className="fas fa-arrow-up" />
            <span>Today ₹{stats.today_revenue.toFixed(0)} · {stats.success_rate.toFixed(1)}% success</span>
          </div>
          <div className="toolbar" style={{ marginBottom: 0, position: 'relative' }}>
            <button className="btn secondary" style={{ background: 'white', color: 'var(--primary-dark)' }} onClick={() => api.downloadPlugin().catch(e => alert(e.message))}>
              <i className="fas fa-download" /> aMember Plugin
            </button>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Revenue by Merchant (30 days)</h3>
          </div>
          <div className="chart-bars">
            {revenue.map(r => (
              <div key={r.merchant_id} className="chart-row">
                <span className="chart-label">{r.merchant_name}</span>
                <div className="chart-bar-wrap">
                  <div className="chart-bar" style={{ width: `${(r.revenue / maxRev) * 100}%` }} />
                </div>
                <span className="chart-value">₹{r.revenue.toFixed(0)}</span>
              </div>
            ))}
            {revenue.length === 0 && <p className="muted">No merchant data yet</p>}
          </div>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card green">
          <div className="stat-label">Today&apos;s Revenue</div>
          <div className="stat-value">₹{stats.today_revenue.toFixed(0)}</div>
          <div className="stat-icon"><i className="fas fa-rupee-sign" /></div>
        </div>
        <div className="stat-card pink">
          <div className="stat-label">Total Transactions</div>
          <div className="stat-value">{stats.total_orders}</div>
          <div className="stat-icon"><i className="fas fa-receipt" /></div>
        </div>
        <div className="stat-card blue">
          <div className="stat-label">Success Rate</div>
          <div className="stat-value">{stats.success_rate.toFixed(1)}%</div>
          <div className="stat-icon"><i className="fas fa-check-circle" /></div>
        </div>
      </div>

      <div className="cards">
        <div className="card"><div className="label">Today orders</div><div className="value">{stats.today_orders}</div></div>
        <div className="card"><div className="label">Today success</div><div className="value green">{stats.today_success}</div></div>
        <div className="card"><div className="label">Pending</div><div className="value">{stats.pending_orders}</div></div>
      </div>
    </>
  )
}

function Orders() {
  const [orders, setOrders] = useState<Order[]>([])
  const [total, setTotal] = useState(0)
  const [status, setStatus] = useState('')
  const [q, setQ] = useState('')

  function load() {
    const params: Record<string, string> = {}
    if (status) params.status = status
    if (q) params.q = q
    api.orders(params).then(r => { setOrders(r.orders); setTotal(r.total) })
  }
  useEffect(() => { load() }, [])

  async function exportCsv() {
    try {
      const params: Record<string, string> = {}
      if (status) params.status = status
      if (q) params.q = q
      await api.exportOrders(params)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Export failed')
    }
  }

  return (
    <>
      <div className="card" style={{ marginBottom: 0 }}>
        <div className="card-header">
          <h3 className="card-title">All Transactions</h3>
          <button className="btn secondary btn-sm" onClick={exportCsv}>
            <i className="fas fa-download" /> Export CSV
          </button>
        </div>
        <div className="toolbar">
          <select className="filter-select" value={status} onChange={e => setStatus(e.target.value)}>
            <option value="">All statuses</option>
            <option value="pending">Pending</option>
            <option value="success">Success</option>
            <option value="failed">Failed</option>
            <option value="expired">Expired</option>
          </select>
          <input placeholder="Search order ID, email…" value={q} onChange={e => setQ(e.target.value)} style={{ flex: 1, minWidth: 200 }} />
          <button className="btn" onClick={load}><i className="fas fa-filter" /> Filter</button>
        </div>
        <p className="muted" style={{ marginBottom: 16 }}>{total} orders</p>
        <div className="table-wrap" style={{ border: 'none', boxShadow: 'none', borderRadius: 0 }}>
        <table>
          <thead>
            <tr>
              <th>Hub ID</th><th>Merchant</th><th>Product</th><th>Amount</th><th>Status</th><th>UTR</th><th>Date</th>
            </tr>
          </thead>
          <tbody>
            {orders.map(o => (
              <tr key={o.id}>
                <td><strong>{o.hub_order_id}</strong><br /><span className="muted">{o.merchant_order_id}</span></td>
                <td>{o.merchant_name}<br /><span className="muted">{o.merchant_domain}</span></td>
                <td>{o.product_name || '—'}<br /><span className="muted">{o.customer_email}</span></td>
                <td>₹{o.pay_amount.toFixed(2)}<br /><span className="muted">base ₹{o.amount}</span></td>
                <td><span className={`badge ${o.status}`}>{o.status}</span></td>
                <td>{o.customer_utr || '—'}</td>
                <td>{new Date(o.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
        </div>
      </div>
    </>
  )
}

function Merchants() {
  const [merchants, setMerchants] = useState<Merchant[]>([])
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [plans, setPlans] = useState<SubscriptionPlan[]>([])
  const [editMerchant, setEditMerchant] = useState<Merchant | null>(null)
  const [merchantSub, setMerchantSub] = useState<SubscriptionUsage | null>(null)
  const [activatePlanId, setActivatePlanId] = useState('')
  const [activateNotes, setActivateNotes] = useState('')
  const [newSecret, setNewSecret] = useState<{ key: string; secret: string } | null>(null)
  const [editForm, setEditForm] = useState({ name: '', domain: '', webhook_url: '', return_url: '', status: 'active' })

  function reload() {
    api.merchants().then(r => setMerchants(r.merchants))
    api.profiles().then(r => setProfiles(r.profiles))
    api.subscriptionPlans().then(r => setPlans(r.plans)).catch(() => {})
  }
  useEffect(() => { reload() }, [])

  async function assignProfile(merchant: Merchant, profileId: string) {
    if (merchant.payment_profile_id && merchant.payment_profile_id !== profileId) {
      const ok = confirm(
        'Profile change: pending orders purani profile se verify honge. Naye orders nayi UPI par jayenge. Continue?'
      )
      if (!ok) { reload(); return }
    }
    await api.assignProfile(merchant.id, profileId)
    reload()
  }

  function openEdit(m: Merchant) {
    setEditMerchant(m)
    setEditForm({
      name: m.name,
      domain: m.domain,
      webhook_url: m.webhook_url,
      return_url: m.return_url || '',
      status: m.status,
    })
    setActivatePlanId('')
    setActivateNotes('')
    setMerchantSub(null)
    api.merchantSubscription(m.id).then(r => setMerchantSub(r.subscription)).catch(() => {})
  }

  async function activatePlan(e: React.FormEvent) {
    e.preventDefault()
    if (!editMerchant || !activatePlanId) return
    try {
      const res = await api.activateSubscription(editMerchant.id, { plan_id: activatePlanId, notes: activateNotes })
      setMerchantSub(res.subscription)
      setActivateNotes('')
      alert('Plan activated')
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  async function saveEdit(e: React.FormEvent) {
    e.preventDefault()
    if (!editMerchant) return
    try {
      await api.updateMerchant(editMerchant.id, editForm)
      setEditMerchant(null)
      reload()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  async function regenSecret() {
    if (!editMerchant || !confirm('Purana secret invalid ho jayega. aMember settings update karni hogi. Continue?')) return
    try {
      const res = await api.regenerateSecret(editMerchant.id)
      setNewSecret({ key: res.api_key, secret: res.api_secret })
      setEditMerchant(null)
      reload()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  return (
    <>
      <div className="toolbar">
        <NavLink to="/websites/add" className="btn" style={{ textDecoration: 'none' }}>
          <i className="fas fa-plus" /> Add Website
        </NavLink>
      </div>

      {newSecret && (
        <div className="form" style={{ marginBottom: 20, maxWidth: 640 }}>
          <strong>New API secret — abhi save karo:</strong>
          <div className="secret-box">
            <div className="copy-row">
              <span>Key: <code>{newSecret.key}</code></span>
              <button type="button" className="btn secondary btn-sm" onClick={() => copyText(newSecret.key)}>Copy</button>
            </div>
          </div>
          <div className="secret-box">
            <div className="copy-row">
              <span>Secret: <code>{newSecret.secret}</code></span>
              <button type="button" className="btn secondary btn-sm" onClick={() => copyText(newSecret.secret)}>Copy</button>
            </div>
          </div>
          <button className="btn secondary" onClick={() => setNewSecret(null)}>Dismiss</button>
        </div>
      )}

      {editMerchant && (
        <div className="modal-backdrop" onClick={() => setEditMerchant(null)}>
          <form className="form modal" onSubmit={saveEdit} onClick={e => e.stopPropagation()}>
            <h3>Edit — {editMerchant.name}</h3>
            <label>Name</label>
            <input value={editForm.name} onChange={e => setEditForm({ ...editForm, name: e.target.value })} required />
            <label>Domain</label>
            <input value={editForm.domain} onChange={e => setEditForm({ ...editForm, domain: e.target.value })} required />
            <label>Webhook URL</label>
            <input value={editForm.webhook_url} onChange={e => setEditForm({ ...editForm, webhook_url: e.target.value })} />
            <label>Status</label>
            <select value={editForm.status} onChange={e => setEditForm({ ...editForm, status: e.target.value })}>
              <option value="active">active</option>
              <option value="suspended">suspended</option>
            </select>
            {merchantSub && (
              <div className="muted" style={{ marginTop: 16, padding: 12, background: '#f8fafc', borderRadius: 8 }}>
                <strong>Subscription:</strong> {merchantSub.plan_name}
                {' '}({merchantSub.orders_used}/{merchantSub.order_limit} orders, {merchantSub.days_left}d left)
              </div>
            )}
            <fieldset style={{ marginTop: 16, border: '1px solid #e2e8f0', borderRadius: 8, padding: 12 }}>
              <legend>Activate plan (manual billing)</legend>
              <label>Plan</label>
              <select value={activatePlanId} onChange={e => setActivatePlanId(e.target.value)}>
                <option value="">— select plan —</option>
                {plans.map(p => (
                  <option key={p.id} value={p.id}>{p.name} — ₹{p.price_inr} ({p.order_limit.toLocaleString()} orders)</option>
                ))}
              </select>
              <label style={{ marginTop: 8 }}>Notes (UTR, payment ref)</label>
              <input value={activateNotes} onChange={e => setActivateNotes(e.target.value)} placeholder="UPI UTR / payment note" />
              <button className="btn secondary" type="button" style={{ marginTop: 8 }} onClick={activatePlan} disabled={!activatePlanId}>
                Activate plan
              </button>
            </fieldset>
            <div className="muted" style={{ marginTop: 12 }}>
              API Key: <code>{editMerchant.api_key}</code>
              <button type="button" className="btn secondary btn-sm" style={{ marginLeft: 8 }} onClick={() => copyText(editMerchant.api_key)}>Copy</button>
            </div>
            <div className="actions">
              <button className="btn" type="submit">Save</button>
              <button className="btn secondary" type="button" onClick={regenSecret}>Regenerate secret</button>
              <button className="btn secondary" type="button" onClick={() => setEditMerchant(null)}>Cancel</button>
            </div>
          </form>
        </div>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Name</th><th>Domain</th><th>API Key</th><th>Status</th><th>UPI Profile</th><th></th></tr>
          </thead>
          <tbody>
            {merchants.map(m => (
              <tr key={m.id}>
                <td><strong>{m.name}</strong></td>
                <td>{m.domain}</td>
                <td>
                  <code>{m.api_key}</code>
                  <button type="button" className="btn secondary btn-sm" style={{ marginLeft: 6 }} onClick={() => copyText(m.api_key)}>Copy</button>
                </td>
                <td><span className={`badge ${m.status === 'active' ? 'success' : 'failed'}`}>{m.status}</span></td>
                <td>
                  <select
                    value={m.payment_profile_id || ''}
                    onChange={e => assignProfile(m, e.target.value)}
                  >
                    <option value="">— select —</option>
                    {profiles.map(p => <option key={p.id} value={p.id}>{p.name} ({p.upi_id})</option>)}
                  </select>
                </td>
                <td><button className="btn secondary btn-sm" onClick={() => openEdit(m)}>Edit</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

function Profiles() {
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [parserTypes, setParserTypes] = useState<ParserType[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState('')
  const [parseResult, setParseResult] = useState('')
  const [parseBody, setParseBody] = useState('')
  const empty = { name: '', upi_id: '', payee_name: 'UPIPays', imap_host: 'imap.gmail.com', imap_port: 993, imap_user: '', imap_password: '', sender_filter: 'hdfcbank', parser_type: 'hdfc', is_active: true }
  const [form, setForm] = useState(empty)

  function reload() { api.profiles().then(r => setProfiles(r.profiles)) }
  useEffect(() => {
    reload()
    api.parserTypes().then(r => setParserTypes(r.parser_types)).catch(() => {})
  }, [])

  function onParserChange(parserType: string) {
    const pt = parserTypes.find(p => p.id === parserType)
    setForm(f => ({
      ...f,
      parser_type: parserType,
      sender_filter: pt?.sender_filter || f.sender_filter,
    }))
  }

  function startEdit(p: Profile) {
    setEditId(p.id)
    setForm({ ...p, imap_password: '' })
    setShowForm(true)
    setTestResult('')
    setParseResult('')
    setParseBody('')
  }

  async function save(e: React.FormEvent) {
    e.preventDefault()
    try {
      if (editId) await api.updateProfile(editId, form)
      else await api.createProfile(form)
      setShowForm(false)
      setEditId(null)
      setForm(empty)
      reload()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  async function testIMAP() {
    if (!editId) return
    setTestResult('Testing…')
    try {
      const r = await api.testIMAP(editId)
      setTestResult(r.ok ? `✓ ${r.message}` : `✗ ${r.message}`)
      if (r.subjects?.length) setTestResult(prev => prev + '\nSubjects: ' + r.subjects.join('; '))
      reload()
    } catch (err) {
      setTestResult('✗ ' + (err instanceof Error ? err.message : 'Failed'))
    }
  }

  async function testParse(fetchLatest = false) {
    if (!editId) return
    setParseResult('Parsing…')
    try {
      const r = await api.testParse(editId, { email_body: parseBody, fetch_latest: fetchLatest })
      if (r.matched) setParseResult(`✓ Amount: ₹${r.amount?.toFixed(2)} — UTR: ${r.utr}`)
      else setParseResult('✗ ' + (r.message || 'No match'))
    } catch (err) {
      setParseResult('✗ ' + (err instanceof Error ? err.message : 'Failed'))
    }
  }

  async function pollNow() {
    if (!editId) return
    try {
      const r = await api.triggerPoll(editId)
      alert(r.message)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  function healthLabel(p: Profile) {
    const h = imapHealthDot(p)
    if (h === 'ok') return <span className="health-dot ok" title="IMAP OK" />
    if (h === 'warn') return <span className="health-dot warn" title="Stale or warning" />
    if (h === 'fail') return <span className="health-dot fail" title={p.imap_last_error || 'Failed'} />
    return <span className="health-dot unknown" title="Not tested" />
  }

  return (
    <>
      <p className="muted" style={{ marginBottom: 16 }}>Each profile = UPI ID + Gmail IMAP. Test IMAP before going live.</p>
      <div className="toolbar">
        <button className="btn" onClick={() => { setShowForm(true); setEditId(null); setForm(empty); setTestResult(''); setParseResult('') }}>
          <i className="fas fa-plus" /> Add profile
        </button>
      </div>
      {showForm && (
        <form className="form form-wide" onSubmit={save} style={{ marginBottom: 24 }}>
          <label>Profile name</label>
          <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
          <label>UPI ID</label>
          <input value={form.upi_id} onChange={e => setForm({ ...form, upi_id: e.target.value })} placeholder="name@okaxis" required />
          <label>Payee name</label>
          <input value={form.payee_name} onChange={e => setForm({ ...form, payee_name: e.target.value })} />
          <label>Bank parser</label>
          <select value={form.parser_type} onChange={e => onParserChange(e.target.value)}>
            {(parserTypes.length ? parserTypes : [{ id: 'hdfc', label: 'HDFC', sender_filter: 'hdfcbank', bank_code: 'hdfc' }]).map(pt => (
              <option key={pt.id} value={pt.id}>{pt.label}</option>
            ))}
          </select>
          <label>Gmail / IMAP user</label>
          <input value={form.imap_user} onChange={e => setForm({ ...form, imap_user: e.target.value })} required />
          <label>IMAP app password {editId && '(leave blank to keep)'}</label>
          <input type="password" value={form.imap_password} onChange={e => setForm({ ...form, imap_password: e.target.value })} />
          <label>Sender filter</label>
          <input value={form.sender_filter} onChange={e => setForm({ ...form, sender_filter: e.target.value })} />
          <label className="radio-label">
            <input type="checkbox" checked={form.is_active} onChange={e => setForm({ ...form, is_active: e.target.checked })} />
            Active (inactive = naye orders block)
          </label>

          {editId && (
            <div className="profile-tools">
              <h4>Test tools</h4>
              <div className="actions">
                <button type="button" className="btn secondary" onClick={testIMAP}>Test IMAP</button>
                <button type="button" className="btn secondary" onClick={() => testParse(true)}>Parse latest email</button>
                <button type="button" className="btn secondary" onClick={pollNow}>Poll now</button>
              </div>
              {testResult && <pre className="test-output">{testResult}</pre>}
              <label style={{ marginTop: 12 }}>Or paste sample email:</label>
              <textarea rows={4} value={parseBody} onChange={e => setParseBody(e.target.value)} placeholder="Paste bank alert email body…" />
              <button type="button" className="btn secondary btn-sm" style={{ marginTop: 8 }} onClick={() => testParse(false)}>Test Parse</button>
              {parseResult && <pre className="test-output">{parseResult}</pre>}
            </div>
          )}

          <div className="actions">
            <button className="btn" type="submit">{editId ? 'Update' : 'Create'}</button>
            <button className="btn secondary" type="button" onClick={() => setShowForm(false)}>Cancel</button>
          </div>
        </form>
      )}
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th></th><th>Name</th><th>UPI ID</th><th>IMAP</th><th>Parser</th><th>Active</th><th></th></tr>
          </thead>
          <tbody>
            {profiles.map(p => (
              <tr key={p.id}>
                <td>{healthLabel(p)}</td>
                <td><strong>{p.name}</strong></td>
                <td>{p.upi_id}</td>
                <td>{p.imap_user}</td>
                <td>{p.parser_type}</td>
                <td><span className={`badge ${p.is_active ? 'success' : 'failed'}`}>{p.is_active ? 'yes' : 'no'}</span></td>
                <td><button className="btn secondary btn-sm" onClick={() => startEdit(p)}>Edit</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

function PlansManager() {
  const [plans, setPlans] = useState<SubscriptionPlan[]>([])
  const [edit, setEdit] = useState<SubscriptionPlan | null>(null)
  const [creating, setCreating] = useState(false)
  const empty = { slug: '', name: '', price_inr: 499, validity_days: 28, order_limit: 5000, is_recommended: false, sort_order: 0, is_active: true, features_json: '[]' }
  const [form, setForm] = useState(empty)

  function reload() { api.subscriptionPlans().then(r => setPlans(r.plans)).catch(() => {}) }
  useEffect(() => { reload() }, [])

  function openEdit(p: SubscriptionPlan) {
    setEdit(p)
    setCreating(false)
    setForm({ ...p, features_json: (p as SubscriptionPlan & { features_json?: string }).features_json || '[]' })
  }

  function openCreate() {
    setEdit(null)
    setCreating(true)
    setForm(empty)
  }

  async function save(e: React.FormEvent) {
    e.preventDefault()
    try {
      if (creating) await api.createPlan(form)
      else if (edit) await api.updatePlan(edit.id, form)
      setEdit(null)
      setCreating(false)
      reload()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed')
    }
  }

  if (edit || creating) {
    return (
      <form className="form form-wide" onSubmit={save}>
        <h3>{creating ? 'New plan' : 'Edit plan'}</h3>
        <label>Slug</label>
        <input value={form.slug} onChange={e => setForm({ ...form, slug: e.target.value })} required disabled={!!edit && edit.slug === 'trial'} />
        <label>Name</label>
        <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
        <label>Price (INR)</label>
        <input type="number" value={form.price_inr} onChange={e => setForm({ ...form, price_inr: +e.target.value })} />
        <label>Validity (days)</label>
        <input type="number" value={form.validity_days} onChange={e => setForm({ ...form, validity_days: +e.target.value })} />
        <label>Order limit</label>
        <input type="number" value={form.order_limit} onChange={e => setForm({ ...form, order_limit: +e.target.value })} />
        <label>Sort order</label>
        <input type="number" value={form.sort_order} onChange={e => setForm({ ...form, sort_order: +e.target.value })} />
        <label>Features JSON</label>
        <textarea rows={4} value={form.features_json} onChange={e => setForm({ ...form, features_json: e.target.value })} placeholder='[{"text":"5,000 QR requests","included":true}]' />
        <label className="radio-label"><input type="checkbox" checked={form.is_recommended} onChange={e => setForm({ ...form, is_recommended: e.target.checked })} /> Recommended badge</label>
        <label className="radio-label"><input type="checkbox" checked={form.is_active} onChange={e => setForm({ ...form, is_active: e.target.checked })} /> Active</label>
        <div className="actions">
          <button className="btn" type="submit">Save</button>
          <button className="btn secondary" type="button" onClick={() => { setEdit(null); setCreating(false) }}>Cancel</button>
        </div>
      </form>
    )
  }

  return (
    <>
      <div className="toolbar"><button className="btn" onClick={openCreate}><i className="fas fa-plus" /> New plan</button></div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>Name</th><th>Price</th><th>Limit</th><th>Days</th><th>Active</th><th></th></tr></thead>
          <tbody>
            {plans.map(p => (
              <tr key={p.id}>
                <td><strong>{p.name}</strong>{p.is_recommended && <span className="badge success" style={{ marginLeft: 8 }}>Rec</span>}</td>
                <td>₹{p.price_inr}</td>
                <td>{p.order_limit.toLocaleString()}</td>
                <td>{p.validity_days}</td>
                <td><span className={`badge ${p.is_active ? 'success' : 'failed'}`}>{p.is_active ? 'yes' : 'no'}</span></td>
                <td><button className="btn secondary btn-sm" onClick={() => openEdit(p)}>Edit</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

function PagesManager() {
  const [pages, setPages] = useState<CMSPage[]>([])
  const [edit, setEdit] = useState<CMSPage | null>(null)
  const [creating, setCreating] = useState(false)
  const empty = { slug: '', title: '', meta_description: '', body_html: '<p>Content here</p>', status: 'draft', show_in_nav: false, nav_label: '', sort_order: 0 }
  const [form, setForm] = useState(empty)

  function reload() { api.cmsPages().then(r => setPages(r.pages)).catch(() => {}) }
  useEffect(() => { reload() }, [])

  function openEdit(p: CMSPage) { setEdit(p); setCreating(false); setForm({ ...p }) }
  function openCreate() { setEdit(null); setCreating(true); setForm(empty) }

  async function save(e: React.FormEvent) {
    e.preventDefault()
    try {
      if (creating) await api.createCMSPage(form)
      else if (edit) await api.updateCMSPage(edit.id, form)
      setEdit(null); setCreating(false); reload()
    } catch (err) { alert(err instanceof Error ? err.message : 'Failed') }
  }

  async function remove(id: string) {
    if (!confirm('Delete this page?')) return
    await api.deleteCMSPage(id)
    reload()
  }

  async function preview(id: string) {
    await api.previewCMSPage(id)
  }

  if (edit || creating) {
    return (
      <form className="form form-wide" onSubmit={save}>
        <h3>{creating ? 'New page' : 'Edit page'}</h3>
        <label>Slug (URL: /slug)</label>
        <input value={form.slug} onChange={e => setForm({ ...form, slug: e.target.value })} placeholder="about" required />
        <label>Title</label>
        <input value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} required />
        <label>Meta description</label>
        <input value={form.meta_description} onChange={e => setForm({ ...form, meta_description: e.target.value })} />
        <label>Body HTML</label>
        <textarea rows={12} value={form.body_html} onChange={e => setForm({ ...form, body_html: e.target.value })} />
        <label>Status</label>
        <select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>
          <option value="draft">draft</option>
          <option value="published">published</option>
        </select>
        <label className="radio-label"><input type="checkbox" checked={form.show_in_nav} onChange={e => setForm({ ...form, show_in_nav: e.target.checked })} /> Show in site nav</label>
        <label>Nav label</label>
        <input value={form.nav_label} onChange={e => setForm({ ...form, nav_label: e.target.value })} />
        <label>Sort order</label>
        <input type="number" value={form.sort_order} onChange={e => setForm({ ...form, sort_order: +e.target.value })} />
        <div className="actions">
          <button className="btn" type="submit">Save</button>
          {!creating && edit && <button className="btn secondary" type="button" onClick={() => preview(edit.id)}>Preview</button>}
          <button className="btn secondary" type="button" onClick={() => { setEdit(null); setCreating(false) }}>Cancel</button>
        </div>
      </form>
    )
  }

  return (
    <>
      <div className="toolbar"><button className="btn" onClick={openCreate}><i className="fas fa-plus" /> New page</button></div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>Title</th><th>Slug</th><th>Status</th><th>Nav</th><th></th></tr></thead>
          <tbody>
            {pages.map(p => (
              <tr key={p.id}>
                <td><strong>{p.title}</strong></td>
                <td>/{p.slug}</td>
                <td><span className={`badge ${p.status === 'published' ? 'success' : 'pending'}`}>{p.status}</span></td>
                <td>{p.show_in_nav ? 'yes' : '—'}</td>
                <td>
                  <button className="btn secondary btn-sm" onClick={() => openEdit(p)}>Edit</button>
                  {' '}
                  <button className="btn secondary btn-sm" onClick={() => preview(p.id)}>Preview</button>
                  {' '}
                  <button className="btn secondary btn-sm" onClick={() => remove(p.id)}>Delete</button>
                </td>
              </tr>
            ))}
            {pages.length === 0 && <tr><td colSpan={5} className="muted">No pages yet</td></tr>}
          </tbody>
        </table>
      </div>
    </>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Shell />}>
        <Route index element={<Dashboard />} />
        <Route path="orders" element={<Orders />} />
        <Route path="unmatched" element={<Unmatched />} />
        <Route path="webhooks" element={<Webhooks />} />
        <Route path="websites/add" element={<AddWebsite />} />
        <Route path="merchants" element={<Merchants />} />
        <Route path="profiles" element={<Profiles />} />
        <Route path="plans" element={<PlansManager />} />
        <Route path="pages" element={<PagesManager />} />
      </Route>
    </Routes>
  )
}
