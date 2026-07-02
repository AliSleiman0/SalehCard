import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { Icon, PageHead, Art, LoadingSpinner, ErrorState, EmptyState, artForCategory } from '@/components'
import { useInventory, useUploadCodes, useLookupCode, useSetThreshold, useUploadHistory } from '../hooks/useInventory'
import { parseCodesFile, type UploadItem } from '../api/codes'
import { relativeTime } from '@/lib/utils'
import type { InventoryStats } from '@/types'

type Tab = 'stock' | 'upload' | 'config' | 'audit'

export default function InventoryPage() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const uploadFor = params.get('upload') ?? undefined
  const [tab, setTab] = useState<Tab>(uploadFor ? 'upload' : 'stock')

  const { data, isLoading, isError, refetch } = useInventory()
  const stats: InventoryStats[] = data?.data ?? []

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_catalog'), t('nav_inventory')]}
        title="Inventory & code management"
        sub="Mission-critical — if codes run out, purchases fail."
      >
        <button className="abtn primary" onClick={() => setTab('upload')}>
          <Icon name="upload" size={15} /> Bulk upload codes
        </button>
      </PageHead>

      <div className="atabs">
        {(
          [
            ['stock', 'Code stock'],
            ['upload', 'Bulk upload'],
            ['config', 'Low-stock config'],
            ['audit', 'Code lookup'],
          ] as [Tab, string][]
        ).map(([k, l]) => (
          <button key={k} className={tab === k ? 'on' : ''} onClick={() => setTab(k)}>
            {l}
          </button>
        ))}
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Could not load inventory" onRetry={() => refetch()} />
      ) : (
        <>
          {tab === 'stock' && <StockTab stats={stats} onAdd={() => setTab('upload')} />}
          {tab === 'upload' && <UploadTab stats={stats} defaultProduct={uploadFor} />}
          {tab === 'config' && <ConfigTab stats={stats} />}
          {tab === 'audit' && <AuditTab />}
        </>
      )}
    </div>
  )
}

function StockTab({ stats, onAdd }: { stats: InventoryStats[]; onAdd: () => void }) {
  const { t } = useTranslation()
  const [lowOnly, setLowOnly] = useState(false)
  const totals = useMemo(() => {
    const uploaded = stats.reduce((a, s) => a + s.uploaded, 0)
    const available = stats.reduce((a, s) => a + s.available, 0)
    const delivered = stats.reduce((a, s) => a + s.delivered, 0)
    const low = stats.filter((s) => s.level === 'lo').length
    return { uploaded, available, delivered, low }
  }, [stats])
  const rows = lowOnly ? stats.filter((s) => s.level === 'lo') : stats

  if (stats.length === 0)
    return <EmptyState icon="layers" title="No code inventory yet" sub="Code-type products will appear here once created." />

  return (
    <div>
      <div className="kpigrid" style={{ gridTemplateColumns: 'repeat(4,1fr)' }}>
        {(
          [
            ['Total codes', totals.uploaded.toLocaleString(), 'box', 'var(--grad)'],
            ['Available', totals.available.toLocaleString(), 'checkc', 'linear-gradient(135deg,#2fd47a,#22e3c8)'],
            ['Delivered', totals.delivered.toLocaleString(), 'send', 'linear-gradient(135deg,#3b5bff,#8a3bff)'],
            ['Low-stock products', String(totals.low), 'alert', 'rgba(255,77,109,.2)'],
          ] as [string, string, 'box' | 'checkc' | 'send' | 'alert', string][]
        ).map(([l, v, ic, bg]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: bg }}>
                <Icon name={ic} size={16} />
              </div>
              <div className="k-label">{l}</div>
            </div>
            <div className="k-val">{v}</div>
          </div>
        ))}
      </div>
      <div className="acard">
        <div className="panelhead">
          <Icon name="layers" size={17} />
          <h3>Code inventory by product</h3>
          <div className="ph-act">
            <div className="aseg">
              <button className={!lowOnly ? 'on' : ''} onClick={() => setLowOnly(false)}>
                {t('all')}
              </button>
              <button className={lowOnly ? 'on' : ''} onClick={() => setLowOnly(true)}>
                Low only
              </button>
            </div>
          </div>
        </div>
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Product</th>
                <th>Uploaded</th>
                <th>Available</th>
                <th>Delivered</th>
                <th>Expired</th>
                <th>Threshold</th>
                <th>Stock level</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((it) => {
                const pct = Math.min(100, Math.round((it.available / Math.max(1, it.threshold * 1.5)) * 100))
                return (
                  <tr key={it.productId}>
                    <td>
                      <div className="cellprod">
                        <Art art={artForCategory(it.category || it.title)} size={34} />
                        <div className="pn">
                          <b>{it.title}</b>
                          <span className="mono">{it.productId.slice(-8)}</span>
                        </div>
                      </div>
                    </td>
                    <td className="num muted">{it.uploaded.toLocaleString()}</td>
                    <td className="num strong">{it.available}</td>
                    <td className="num muted">{it.delivered.toLocaleString()}</td>
                    <td className="num faint">{it.expired}</td>
                    <td className="num muted">{it.threshold}</td>
                    <td>
                      <span className={'stock ' + it.level}>
                        <div className="bar">
                          <i style={{ width: (it.available === 0 ? 4 : pct) + '%' }} />
                        </div>
                        <span className="num">
                          {it.available === 0 ? 'Empty' : it.level === 'lo' ? 'Low' : it.level === 'mid' ? 'OK' : 'Healthy'}
                        </span>
                      </span>
                    </td>
                    <td>
                      <div className="row-actions">
                        <button className="abtn xs" onClick={onAdd}>
                          <Icon name="upload" size={13} /> Add
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

function UploadTab({ stats, defaultProduct }: { stats: InventoryStats[]; defaultProduct?: string }) {
  const upload = useUploadCodes()
  const { data: historyRes } = useUploadHistory()
  const history = historyRes?.data ?? []
  const fileRef = useRef<HTMLInputElement>(null)
  const [productId, setProductId] = useState(defaultProduct || stats[0]?.productId || '')
  const [fileName, setFileName] = useState('')
  const [items, setItems] = useState<UploadItem[]>([])
  const [invalid, setInvalid] = useState(0)
  const [result, setResult] = useState<{ inserted: number; duplicates: number; invalid: number } | null>(null)

  // Duplicate detection within the parsed batch (server also dedupes vs existing).
  const dupes = useMemo(() => {
    const seen = new Set<string>()
    let d = 0
    for (const it of items) {
      if (seen.has(it.code)) d++
      else seen.add(it.code)
    }
    return d
  }, [items])
  const validCount = items.length - dupes

  const onFile = (file: File) => {
    setResult(null)
    file.text().then((text) => {
      const parsed = parseCodesFile(text)
      setItems(parsed.items)
      setInvalid(parsed.invalid)
      setFileName(file.name)
    })
  }

  const commit = () => {
    if (!productId || items.length === 0) return
    upload.mutate(
      { productId, codes: items },
      {
        onSuccess: (res) => {
          setResult(res.data ?? null)
          setItems([])
          setFileName('')
        },
      }
    )
  }

  return (
    <div className="formgrid">
      <div className="fieldset">
        <div className="acard pad">
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Bulk upload codes</h3>
          <label className="alabel">Product</label>
          <select
            className="select"
            style={{ width: '100%', marginBottom: 16 }}
            value={productId}
            onChange={(e) => setProductId(e.target.value)}
          >
            {stats.length === 0 && <option value="">No code products</option>}
            {stats.map((it) => (
              <option key={it.productId} value={it.productId}>
                {it.title} — {it.available} available
              </option>
            ))}
          </select>
          <input
            ref={fileRef}
            type="file"
            accept=".csv,.txt"
            style={{ display: 'none' }}
            onChange={(e) => e.target.files?.[0] && onFile(e.target.files[0])}
          />
          <div className="dropzone">
            <div className="dz-ic">
              <Icon name="upload" size={22} />
            </div>
            <div style={{ fontWeight: 800, fontSize: 14, color: 'var(--text)' }}>Drop a CSV or TXT file of codes</div>
            <div className="ahint" style={{ marginTop: 6 }}>
              One code per line, or <span style={{ fontFamily: 'ui-monospace' }}>code,pin</span> per line · max 50,000
            </div>
            <button className="abtn sm" style={{ marginTop: 14 }} onClick={() => fileRef.current?.click()}>
              Choose file
            </button>
          </div>

          {fileName && (
            <div style={{ marginTop: 16, padding: 14, background: 'var(--surface-2)', borderRadius: 'var(--ar-sm)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                <Icon name="file" size={15} />
                <b style={{ fontSize: 13 }}>{fileName}</b>
                <span className="faint" style={{ fontSize: 12, marginInlineStart: 'auto' }}>
                  {items.length} rows
                </span>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 10 }}>
                <span className="st st-ok" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {validCount} valid
                </span>
                <span className="st st-warn" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {dupes} duplicates
                </span>
                <span className="st st-mute" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {invalid} bad format
                </span>
              </div>
            </div>
          )}

          {result && (
            <div
              style={{
                marginTop: 16,
                padding: 14,
                background: 'var(--ff-credit-bg)',
                border: '1px solid var(--ff-credit-bd)',
                borderRadius: 'var(--ar-sm)',
                fontSize: 13,
                fontWeight: 700,
              }}
            >
              <Icon name="checkc" size={15} /> Committed {result.inserted} codes ({result.duplicates} duplicates skipped).
            </div>
          )}
          {upload.error && (
            <div style={{ marginTop: 12, color: 'var(--danger)', fontSize: 13, fontWeight: 700 }}>
              {(upload.error as Error).message}
            </div>
          )}

          <div style={{ display: 'flex', gap: 10, marginTop: 16, justifyContent: 'flex-end' }}>
            <button className="abtn" onClick={() => fileRef.current?.click()}>
              Choose another
            </button>
            <button className="abtn primary" disabled={validCount === 0 || upload.isPending} onClick={commit}>
              <Icon name="check" size={15} /> {upload.isPending ? 'Committing…' : `Commit ${validCount} codes`}
            </button>
          </div>
        </div>
      </div>
      <div className="fieldset">
        <div className="acard">
          <div className="panelhead">
            <Icon name="clock" size={17} />
            <h3>Upload history</h3>
          </div>
          <div>
            {history.length === 0 && (
              <div className="lsrow faint" style={{ fontSize: 12.5 }}>
                No uploads yet — bulk-uploaded code batches will appear here.
              </div>
            )}
            {history.map((u) => (
              <div className="lsrow" key={u.id} style={{ flexDirection: 'column', alignItems: 'stretch', gap: 6 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <b style={{ fontSize: 13 }}>{u.productTitle || u.productId}</b>
                  <span className="num faint" style={{ marginInlineStart: 'auto', fontSize: 12 }}>
                    {relativeTime(u.createdAt)}
                  </span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
                  <span className="st st-ok" style={{ fontSize: 10.5 }}>
                    +{u.inserted} codes
                  </span>
                  {u.duplicates > 0 && (
                    <span className="st st-warn" style={{ fontSize: 10.5 }}>
                      {u.duplicates} dupes
                    </span>
                  )}
                  <span className="faint" style={{ marginInlineStart: 'auto' }}>
                    by {u.uploadedBy ? u.uploadedBy.split('@')[0] : '—'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

function ConfigTab({ stats }: { stats: InventoryStats[] }) {
  const { t } = useTranslation()
  const setThreshold = useSetThreshold()
  const [edits, setEdits] = useState<Record<string, string>>({})

  const save = (productId: string) => {
    const raw = edits[productId]
    const n = parseInt(raw ?? '', 10)
    if (!Number.isNaN(n)) setThreshold.mutate({ productId, threshold: n })
  }

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="alert" size={17} />
        <h3>Low-stock thresholds</h3>
        <div className="ph-act">
          <span className="faint" style={{ fontSize: 12 }}>
            Edit a value and press Enter to save
          </span>
        </div>
      </div>
      <div className="tablewrap">
        <table className="tbl">
          <thead>
            <tr>
              <th>Product</th>
              <th>Available now</th>
              <th>Alert threshold</th>
              <th>Auto-disable at 0</th>
              <th>{t('status')}</th>
            </tr>
          </thead>
          <tbody>
            {stats.map((it) => {
              const value = edits[it.productId] ?? String(it.threshold)
              return (
                <tr key={it.productId}>
                  <td>
                    <div className="cellprod">
                      <Art art={artForCategory(it.category || it.title)} size={32} />
                      <div className="pn">
                        <b>{it.title}</b>
                      </div>
                    </div>
                  </td>
                  <td className="num strong">{it.available}</td>
                  <td>
                    <input
                      className="afield"
                      style={{ padding: '6px 10px', width: 90 }}
                      value={value}
                      onChange={(e) => setEdits({ ...edits, [it.productId]: e.target.value })}
                      onKeyDown={(e) => e.key === 'Enter' && save(it.productId)}
                      onBlur={() => edits[it.productId] !== undefined && save(it.productId)}
                    />
                  </td>
                  <td>
                    <div className={'tog' + (it.level === 'lo' ? ' on' : '')} />
                  </td>
                  <td>
                    {it.available < it.threshold ? (
                      <span className="st st-danger">
                        <i className="d" />
                        Below threshold
                      </span>
                    ) : (
                      <span className="st st-ok">
                        <i className="d" />
                        OK
                      </span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function AuditTab() {
  const [code, setCode] = useState('')
  const { data, isFetching } = useLookupCode(code)
  const audit = data?.data

  return (
    <div className="formgrid">
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Code lookup &amp; audit</h3>
        <div className="fsearch" style={{ minWidth: 0, marginBottom: 18 }}>
          <Icon name="search" size={16} />
          <input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="Paste a code or last 4 digits…"
            style={{ fontFamily: 'ui-monospace' }}
          />
        </div>
        {isFetching && <LoadingSpinner />}
        {!isFetching && code.trim().length >= 3 && !audit && (
          <EmptyState icon="search" title="No matching code" sub="Check the value and try again." />
        )}
        {audit && (
          <>
            <div className="avault" style={{ marginBottom: 18 }}>
              <span className="vc">{audit.code.code}</span>
              <span className="st st-mute">
                <i className="d" />
                {audit.code.status}
              </span>
            </div>
            <div className="deflist">
              <div className="defrow">
                <span className="dk">Product</span>
                <span className="dv">{audit.product?.title ?? audit.code.productId}</span>
              </div>
              <div className="defrow">
                <span className="dk">Code ID</span>
                <span className="dv mono">{audit.code.id.slice(-10)}</span>
              </div>
              <div className="defrow">
                <span className="dk">Status</span>
                <span className="dv" style={{ textTransform: 'capitalize' }}>
                  {audit.code.status}
                </span>
              </div>
              <div className="defrow">
                <span className="dk">Consumed by order</span>
                <span className="dv mono">{audit.code.orderId ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">Delivered to</span>
                <span className="dv">{audit.code.deliveredTo ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">Delivered at</span>
                <span className="dv">{audit.code.deliveredAt ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">Uploaded batch</span>
                <span className="dv">{audit.code.batch ?? '—'}</span>
              </div>
            </div>
          </>
        )}
      </div>
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>Audit trail</h3>
        {!audit ? (
          <div className="faint" style={{ fontSize: 13 }}>
            Look up a code to see its lifecycle.
          </div>
        ) : (
          <div className="timeline">
            {(
              [
                ['done', 'Uploaded', audit.code.batch ?? 'batch', 'checkc'],
                [
                  audit.code.orderId ? 'done' : 'todo',
                  'Reserved at checkout',
                  audit.code.orderId ?? '—',
                  'clock',
                ],
                [
                  audit.code.status === 'delivered' ? 'done' : 'todo',
                  'Delivered to customer',
                  audit.code.deliveredAt ?? '—',
                  'send',
                ],
                [audit.code.status === 'delivered' ? 'active' : 'todo', 'Active in customer vault', '—', 'shield'],
              ] as [string, string, string, 'checkc' | 'clock' | 'send' | 'shield'][]
            ).map(([st, title, sub, ic], i) => (
              <div className={'tlitem ' + st} key={i}>
                <div className="tldot">
                  <Icon name={ic} size={14} />
                </div>
                <div className="tlbody">
                  <b>{title}</b>
                  <span>{sub}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
