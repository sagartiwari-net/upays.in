import { useEffect, useState } from 'react'
import { api, Order, UnmatchedTxn } from './api'

export default function Unmatched() {
  const [items, setItems] = useState<UnmatchedTxn[]>([])
  const [total, setTotal] = useState(0)
  const [orders, setOrders] = useState<Order[]>([])
  const [approveFor, setApproveFor] = useState<UnmatchedTxn | null>(null)
  const [selectedOrder, setSelectedOrder] = useState('')
  const [error, setError] = useState('')

  function reload() {
    api.unmatched().then(r => { setItems(r.unmatched); setTotal(r.total) })
  }
  useEffect(() => { reload() }, [])

  async function openApprove(item: UnmatchedTxn) {
    setApproveFor(item)
    setSelectedOrder('')
    setError('')
    const res = await api.orders({ status: 'pending', limit: '50' })
    setOrders(res.orders.filter(o => Math.abs(o.pay_amount - item.amount) < 0.01))
  }

  async function confirmApprove() {
    if (!approveFor || !selectedOrder) return
    setError('')
    try {
      await api.manualApprove(selectedOrder, {
        utr: approveFor.utr,
        amount: approveFor.amount,
        bank_txn_id: approveFor.id,
      })
      setApproveFor(null)
      reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed')
    }
  }

  return (
    <>
      <p className="muted" style={{ marginBottom: 12 }}>{total} unmatched</p>

      {approveFor && (
        <div className="modal-backdrop" onClick={() => setApproveFor(null)}>
          <div className="form modal" onClick={e => e.stopPropagation()}>
            <h3>Manual approve</h3>
            <p>UTR: <code>{approveFor.utr}</code> — ₹{approveFor.amount.toFixed(2)}</p>
            <label>Pending order (same pay amount)</label>
            <select value={selectedOrder} onChange={e => setSelectedOrder(e.target.value)}>
              <option value="">— select order —</option>
              {orders.map(o => (
                <option key={o.id} value={o.id}>
                  {o.hub_order_id} — {o.merchant_name} — ₹{o.pay_amount.toFixed(2)}
                </option>
              ))}
            </select>
            {orders.length === 0 && <p className="muted">No pending order with pay amount ₹{approveFor.amount.toFixed(2)}</p>}
            {error && <div className="error">{error}</div>}
            <div className="actions">
              <button className="btn" disabled={!selectedOrder} onClick={confirmApprove}>Approve</button>
              <button className="btn secondary" onClick={() => setApproveFor(null)}>Cancel</button>
            </div>
          </div>
        </div>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr><th>UTR</th><th>Amount</th><th>Profile</th><th>Date</th><th>Excerpt</th><th></th></tr>
          </thead>
          <tbody>
            {items.map(item => (
              <tr key={item.id}>
                <td><code>{item.utr}</code></td>
                <td>₹{item.amount.toFixed(2)}</td>
                <td>{item.profile_name || '—'}</td>
                <td>{new Date(item.created_at).toLocaleString()}</td>
                <td className="muted" style={{ fontSize: 12, maxWidth: 200, wordBreak: 'break-all' }}>{item.raw_excerpt?.slice(0, 80)}…</td>
                <td><button className="btn secondary btn-sm" onClick={() => openApprove(item)}>Approve</button></td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr><td colSpan={6} className="muted" style={{ textAlign: 'center' }}>No unmatched payments</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}
