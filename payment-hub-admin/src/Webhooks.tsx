import { useEffect, useState } from 'react'
import { api, Webhook } from './api'

export default function Webhooks() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)

  function load(off = 0) {
    api.webhooks(String(off)).then(r => {
      setWebhooks(r.webhooks)
      setTotal(r.total)
      setOffset(off)
    })
  }
  useEffect(() => { load() }, [])

  return (
    <>
      <p className="muted" style={{ marginBottom: 12 }}>{total} total</p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>Hub Order</th><th>Merchant</th><th>Direction</th><th>Status</th><th>HTTP</th><th>Time</th></tr>
          </thead>
          <tbody>
            {webhooks.map(w => (
              <tr key={w.id}>
                <td><code>{w.hub_order_id || '—'}</code></td>
                <td>{w.merchant_name || '—'}</td>
                <td>{w.direction}</td>
                <td><span className={`badge ${w.status === 'success' ? 'success' : 'failed'}`}>{w.status}</span></td>
                <td>{w.response_code ?? '—'}</td>
                <td>{new Date(w.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {webhooks.length === 0 && (
              <tr><td colSpan={6} className="muted" style={{ textAlign: 'center' }}>No webhook logs yet</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="toolbar" style={{ marginTop: 16 }}>
        <button className="btn secondary" disabled={offset === 0} onClick={() => load(Math.max(0, offset - 25))}>← Prev</button>
        <button className="btn secondary" disabled={offset + 25 >= total} onClick={() => load(offset + 25)}>Next →</button>
      </div>
    </>
  )
}
