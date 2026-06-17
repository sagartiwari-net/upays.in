import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, copyText, Merchant, OnboardWebsiteResult, Profile } from './api'

const emptyProfile = {
  name: '',
  upi_id: '',
  payee_name: 'UPIPays',
  imap_host: 'imap.gmail.com',
  imap_port: 993,
  imap_user: '',
  imap_password: '',
  sender_filter: 'hdfcbank',
  parser_type: 'hdfc',
  is_active: true,
}

export default function AddWebsite() {
  const nav = useNavigate()
  const [step, setStep] = useState(1)
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [error, setError] = useState('')
  const [result, setResult] = useState<OnboardWebsiteResult | null>(null)

  const [site, setSite] = useState({
    name: '',
    domain: '',
    webhook_url: '',
    return_url: '',
  })

  const [payMode, setPayMode] = useState<'existing' | 'new'>('existing')
  const [profileId, setProfileId] = useState('')
  const [profile, setProfile] = useState(emptyProfile)

  useEffect(() => {
    api.profiles().then(r => {
      setProfiles(r.profiles)
      if (r.profiles.length > 0) setProfileId(r.profiles[0].id)
    })
    api.parserTypes().then(r => {
      if (r.parser_types.length > 0 && payMode === 'new') {
        setProfile(p => ({ ...p, parser_type: r.parser_types[0].id, sender_filter: r.parser_types[0].sender_filter }))
      }
    }).catch(() => {})
  }, [])

  function suggestWebhook() {
    const d = site.domain.trim()
    if (!d) return
    const host = d.startsWith('app.') ? d : 'app.' + d
    setSite(s => ({
      ...s,
      webhook_url: `https://${host}/payment/upipays/ipn`,
    }))
  }

  async function submit() {
    setError('')
    try {
      const res = await api.onboardWebsite({
        merchant: site,
        payment:
          payMode === 'existing'
            ? { mode: 'existing', profile_id: profileId }
            : { mode: 'new', profile },
      })
      setResult(res)
      setStep(3)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed')
    }
  }

  if (step === 3 && result) {
    const m = result.merchant
    return (
      <>
        <div className="wizard-success">
          <p className="success-title">✓ {m.name} is ready</p>
          <div className="secret-box">
            <div className="copy-row">
              <span>API Key: <code>{m.api_key}</code></span>
              <button type="button" className="btn secondary btn-sm" onClick={() => copyText(m.api_key)}>Copy</button>
            </div>
          </div>
          <div className="secret-box">
            <div className="copy-row">
              <span>API Secret: <code>{m.api_secret}</code></span>
              <button type="button" className="btn secondary btn-sm" onClick={() => copyText(m.api_secret)}>Copy</button>
            </div>
          </div>
          <p className="muted">Secret sirf ek baar dikhta hai — abhi save kar lo.</p>

          <div className="checklist">
            <h3>Next steps</h3>
            <ol>
              <li>aMember plugin upload: <code>{result.checklist.amember_plugin_path}</code></li>
              <li>Hub URL in plugin: <code>{result.checklist.hub_url}</code></li>
              <li>Webhook URL: <code>{result.checklist.webhook_url}</code></li>
              <li>aMember → UPIPaysment → paste API Key + Secret → Save</li>
              <li>Test ₹1 payment → Transactions page par success dikhe</li>
            </ol>
          </div>

          <div className="actions">
            <button className="btn" onClick={() => nav('/merchants')}>Go to Merchants</button>
            <button className="btn secondary" onClick={() => { setStep(1); setResult(null); setSite({ name: '', domain: '', webhook_url: '', return_url: '' }) }}>
              Add another
            </button>
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <div className="wizard-steps">
        <span className={step === 1 ? 'active' : ''}>1. Site</span>
        <span className={step === 2 ? 'active' : ''}>2. Payment</span>
        <span>3. Done</span>
      </div>

      {step === 1 && (
        <form className="form wizard-form" onSubmit={e => { e.preventDefault(); suggestWebhook(); setStep(2) }}>
          <label>Website / business name</label>
          <input value={site.name} onChange={e => setSite({ ...site, name: e.target.value })} placeholder="Semrush Toolz" required />

          <label>Domain</label>
          <input value={site.domain} onChange={e => setSite({ ...site, domain: e.target.value })} placeholder="semrushtoolz.com" required />

          <label>Webhook URL (aMember IPN)</label>
          <input value={site.webhook_url} onChange={e => setSite({ ...site, webhook_url: e.target.value })} placeholder="https://app.example.com/payment/upipays/ipn" required />
          <button type="button" className="btn secondary btn-sm" style={{ marginTop: 8 }} onClick={suggestWebhook}>
            Auto-fill from domain
          </button>

          <label>Return URL (optional)</label>
          <input value={site.return_url} onChange={e => setSite({ ...site, return_url: e.target.value })} />

          <div className="actions">
            <button className="btn" type="submit">Next →</button>
            <Link to="/merchants" className="btn secondary" style={{ display: 'inline-flex', alignItems: 'center' }}>Cancel</Link>
          </div>
        </form>
      )}

      {step === 2 && (
        <div className="form wizard-form">
          <label>Payment setup</label>
          <div className="radio-group">
            <label className="radio-label">
              <input type="radio" checked={payMode === 'existing'} onChange={() => setPayMode('existing')} />
              Use existing UPI profile (same bank account)
            </label>
            <label className="radio-label">
              <input type="radio" checked={payMode === 'new'} onChange={() => setPayMode('new')} />
              New UPI + Gmail (alag bank / email)
            </label>
          </div>

          {payMode === 'existing' && (
            <>
              <label>Select profile</label>
              <select value={profileId} onChange={e => setProfileId(e.target.value)}>
                {profiles.map(p => (
                  <option key={p.id} value={p.id}>{p.name} — {p.upi_id}</option>
                ))}
              </select>
              {profiles.length === 0 && <p className="error">No profiles yet. Choose "New UPI" or add from UPI Profiles page.</p>}
            </>
          )}

          {payMode === 'new' && (
            <>
              <label>Profile name</label>
              <input value={profile.name} onChange={e => setProfile({ ...profile, name: e.target.value })} placeholder="Site B HDFC" required />
              <label>UPI ID</label>
              <input value={profile.upi_id} onChange={e => setProfile({ ...profile, upi_id: e.target.value })} placeholder="name@okaxis" required />
              <label>Payee name (QR par dikhega)</label>
              <input value={profile.payee_name} onChange={e => setProfile({ ...profile, payee_name: e.target.value })} />
              <label>Gmail / IMAP email</label>
              <input value={profile.imap_user} onChange={e => setProfile({ ...profile, imap_user: e.target.value })} required />
              <label>Gmail App Password</label>
              <input type="password" value={profile.imap_password} onChange={e => setProfile({ ...profile, imap_password: e.target.value })} required />
              <label>Bank parser</label>
              <select value={profile.parser_type} onChange={e => {
                const v = e.target.value
                const filters: Record<string, string> = { hdfc: 'hdfcbank', sbi: 'sbi', icici: 'icicibank', axis: 'axisbank', generic: '' }
                setProfile({ ...profile, parser_type: v, sender_filter: filters[v] ?? profile.sender_filter })
              }}>
                <option value="hdfc">HDFC Bank</option>
                <option value="sbi">SBI Bank</option>
                <option value="icici">ICICI Bank</option>
                <option value="axis">Axis Bank</option>
                <option value="generic">Generic</option>
              </select>
              <label>Email sender filter</label>
              <input value={profile.sender_filter} onChange={e => setProfile({ ...profile, sender_filter: e.target.value })} placeholder="hdfcbank" />
            </>
          )}

          {error && <div className="error">{error}</div>}

          <div className="actions">
            <button className="btn secondary" type="button" onClick={() => setStep(1)}>← Back</button>
            <button className="btn" type="button" onClick={submit} disabled={payMode === 'existing' && !profileId}>
              Create website
            </button>
          </div>
        </div>
      )}
    </>
  )
}
