import { useEffect, useMemo, useRef, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { Icon, PageHead, Art, LoadingSpinner, ErrorState, EmptyState, Pagination, artForCategory } from '@/components'
import {
  useInventory,
  useInventoryPage,
  useUploadCodes,
  useLookupCode,
  useSetThreshold,
  useUploadHistory,
} from '../hooks/useInventory'
import { parseCodesFile, listCodes, type UploadItem } from '../api/codes'
import { CodesModal } from '../components/CodesModal'
import { formatDateTime, downloadCsv, money } from '@/lib/utils'
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import { useKeyedPage } from '@/hooks/useKeyedPage'
import type { InventoryStats, Code } from '@/types'

type Tab = 'stock' | 'upload' | 'config' | 'audit'

export default function InventoryPage() {
  const { t } = useTranslation()
  const can = useCan()
  const canManage = can('inventory.manage')
  const [params, setSearchParams] = useSearchParams()
  const uploadFor = params.get('upload') ?? undefined
  const [tab, setTab] = useState<Tab>(uploadFor ? 'upload' : 'stock')
  // Which product the Bulk-upload picker should preselect. Driven by the
  // ?upload=<id> URL param (a shareable deep link) so clicking a Code-stock row
  // navigates there and the picker follows.
  const [uploadProduct, setUploadProduct] = useState<string | undefined>(uploadFor)
  const openUpload = (productId?: string) => {
    // Set/clear only ?upload=, keeping the stock view (?view=) so going back to
    // the Code stock tab still shows the list the admin came from.
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (productId) next.set('upload', productId)
        else next.delete('upload')
        return next
      },
      { replace: !productId },
    )
    // Switch directly too: the effect below only fires when ?upload= changes,
    // so re-clicking the same row after visiting another tab would do nothing.
    if (productId) setUploadProduct(productId)
    setTab('upload')
  }
  // React to ?upload= changes (deep link, or a row click that set the param
  // while already mounted — useState alone only reads the first value).
  useEffect(() => {
    if (uploadFor) {
      setUploadProduct(uploadFor)
      setTab('upload')
    }
  }, [uploadFor])

  // The Code-stock table paginates from the backend (see StockTab). The upload
  // picker and thresholds editor need every product, so they fetch the full list
  // — gated to those tabs so the stock view doesn't also download everything.
  const needsFullList = tab === 'upload' || tab === 'config'
  const { data, isLoading, isError, refetch } = useInventory(needsFullList)
  const stats: InventoryStats[] = data?.data ?? []

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_catalog'), t('nav_inventory')]}
        title={t('inv_title')}
        sub={t('inv_sub')}
      >
        {canManage && (
          <button className="abtn primary" onClick={() => openUpload()}>
            <Icon name="upload" size={15} /> {t('inv_bulk_upload_codes')}
          </button>
        )}
      </PageHead>

      <div className="atabs">
        {(
          [
            ['stock', t('inv_tab_stock')],
            ['upload', t('inv_tab_upload')],
            ['config', t('inv_tab_config')],
            ['audit', t('inv_tab_audit')],
          ] as [Tab, string][]
        )
          // Upload + threshold config are mutations — hidden for view-only admins.
          .filter(([k]) => canManage || (k !== 'upload' && k !== 'config'))
          .map(([k, l]) => (
          <button key={k} className={tab === k ? 'on' : ''} onClick={() => setTab(k)}>
            {l}
          </button>
        ))}
      </div>

      {tab === 'stock' && <StockTab onAdd={openUpload} />}
      {tab === 'audit' && <AuditTab />}
      {needsFullList &&
        (isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorState message={t('inv_load_error')} onRetry={() => refetch()} />
        ) : (
          <>
            {tab === 'upload' && canManage && <UploadTab stats={stats} defaultProduct={uploadProduct} />}
            {tab === 'config' && canManage && <ConfigTab stats={stats} />}
          </>
        ))}
    </div>
  )
}

const PAGE_SIZE = 20

type StockView = 'all' | 'below' | 'low'

function StockTab({ onAdd }: { onAdd: (productId?: string) => void }) {
  const { t } = useTranslation()
  const canManage = useCan()('inventory.manage')
  // The stock view lives in the URL so the dashboard's Low-stock tile can
  // deep-link to it and a refresh keeps it:
  //   ?view=below  every product under its threshold (the tile's own count)
  //   ?view=low    only the most depleted ("lo" level — the KPI card below)
  const [params, setSearchParams] = useSearchParams()
  const view: StockView = params.get('view') === 'below' ? 'below' : params.get('view') === 'low' ? 'low' : 'all'
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  // Debounce the search box.
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(timer)
  }, [search])
  // A view change or new search starts at page 1, in the same render.
  const [page, setPage] = useKeyedPage(`${view}|${debouncedSearch.trim()}`)
  // Backend-paginated: one page of rows + global KPI totals in meta.totals.
  const { data, isLoading, isError, refetch } = useInventoryPage({
    page,
    limit: PAGE_SIZE,
    low: view === 'low',
    below: view === 'below',
    q: debouncedSearch.trim(),
  })
  const rows = data?.data ?? []
  const meta = data?.meta
  const totals = meta?.totals
  const [exportingId, setExportingId] = useState('')
  // Which product's codes popup is open (null = closed).
  const [codesFor, setCodesFor] = useState<InventoryStats | null>(null)

  // Switch the stock view (page resets via useKeyedPage). Drops any stale
  // ?upload= left over from a row click.
  const setView = (v: StockView) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (v === 'all') next.delete('view')
        else next.set('view', v)
        next.delete('upload')
        return next
      },
      { replace: true },
    )
  }

  // Export a single product's code list (all statuses, across all pages — the
  // backend caps limit at 100, so page to total).
  const exportCodes = async (productId: string, title: string) => {
    if (exportingId) return
    setExportingId(productId)
    try {
      const all: Code[] = []
      let p = 1
      let pages = 1
      do {
        const res = await listCodes(productId, { page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = [
        t('inv_f_code'),
        t('inv_f_pin'),
        t('status'),
        t('inv_f_order_id'),
        t('inv_f_delivered_to'),
        t('inv_f_delivered_at'),
        t('inv_f_batch'),
      ]
      const csvRows = all.map((c) => [
        c.code,
        c.pin ?? '',
        c.status,
        c.orderId ?? '',
        c.deliveredTo ?? '',
        c.deliveredAt ?? '',
        c.batch ?? '',
      ])
      const safe = title.replace(/[^a-z0-9]+/gi, '-').toLowerCase()
      downloadCsv(`codes-${safe}-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExportingId('')
    }
  }

  if (isLoading) return <LoadingSpinner />
  if (isError) return <ErrorState message={t('inv_load_error')} onRetry={() => refetch()} />
  if (view === 'all' && !debouncedSearch.trim() && (meta?.total ?? 0) === 0)
    return <EmptyState icon="layers" title={t('inv_empty_title')} sub={t('inv_empty_sub')} />

  return (
    <div>
      <div className="kpigrid" style={{ gridTemplateColumns: 'repeat(5,1fr)' }}>
        {(
          [
            [t('inv_kpi_total_codes'), (totals?.uploaded ?? 0).toLocaleString(), 'box', 'var(--grad)'],
            [t('inv_available'), (totals?.available ?? 0).toLocaleString(), 'checkc', 'linear-gradient(135deg,#2fd47a,#22e3c8)'],
            [t('inv_delivered'), (totals?.delivered ?? 0).toLocaleString(), 'send', 'linear-gradient(135deg,#3b5bff,#8a3bff)'],
            [t('inv_kpi_low_stock'), String(totals?.lowStock ?? 0), 'alert', 'rgba(255,77,109,.2)'],
            [
              t('inv_kpi_stock_value'),
              money(totals?.totalValue ?? 0),
              'wallet',
              'linear-gradient(135deg,#f0a020,#f5c542)',
              totals?.unvaluedProducts
                ? t('inv_unpriced', { count: totals.unvaluedProducts })
                : undefined,
            ],
          ] as [string, string, 'box' | 'checkc' | 'send' | 'alert' | 'wallet', string, string?][]
        ).map(([l, v, ic, bg, note]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: bg }}>
                <Icon name={ic} size={16} />
              </div>
              <div className="k-label">{l}</div>
            </div>
            <div className="k-val">{v}</div>
            {note && (
              <div className="k-foot" title={t('inv_unpriced_tip')}>
                <span className="k-since">{note}</span>
              </div>
            )}
          </div>
        ))}
      </div>
      <div className="acard">
        <div className="panelhead">
          <Icon name="layers" size={17} />
          <h3>{t('inv_by_product')}</h3>
          <div className="ph-act">
            <div className="fsearch" style={{ minWidth: 0 }}>
              <Icon name="search" size={15} />
              <input
                placeholder={t('inv_search_ph')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div className="aseg">
              <button className={view === 'all' ? 'on' : ''} onClick={() => setView('all')}>
                {t('all')}
              </button>
              <button className={view === 'below' ? 'on' : ''} onClick={() => setView('below')}>
                {t('inv_below_threshold')}
              </button>
              <button className={view === 'low' ? 'on' : ''} onClick={() => setView('low')}>
                {t('inv_low_only')}
              </button>
            </div>
          </div>
        </div>
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>{t('col_product')}</th>
                <th>{t('inv_uploaded')}</th>
                <th>{t('inv_available')}</th>
                <th>{t('inv_delivered')}</th>
                <th>{t('inv_expired')}</th>
                <th>{t('inv_unit_price')}</th>
                <th>{t('inv_total_value')}</th>
                <th>{t('inv_threshold')}</th>
                <th>{t('inv_stock_level')}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((it) => {
                const pct = Math.min(100, Math.round((it.available / Math.max(1, it.threshold * 1.5)) * 100))
                return (
                  <tr key={it.productId}>
                    <td>
                      <div
                        className="cellprod"
                        onClick={() => canManage && onAdd(it.productId)}
                        style={{ cursor: canManage ? 'pointer' : 'default' }}
                        title={canManage ? t('inv_row_upload_tip') : undefined}
                      >
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
                    <td className="num muted">{it.unitPrice != null ? money(it.unitPrice) : '—'}</td>
                    <td className="num strong">{it.totalValue != null ? money(it.totalValue) : '—'}</td>
                    <td className="num muted">{it.threshold}</td>
                    <td>
                      <span className={'stock ' + it.level}>
                        <div className="bar">
                          <i style={{ width: (it.available === 0 ? 4 : pct) + '%' }} />
                        </div>
                        <span className="num">
                          {it.available === 0
                            ? t('inv_level_empty')
                            : it.level === 'lo'
                              ? t('inv_level_low')
                              : it.level === 'mid'
                                ? t('inv_level_ok')
                                : t('inv_level_healthy')}
                        </span>
                      </span>
                    </td>
                    <td>
                      <div className="row-actions">
                        {canManage && (
                          <button
                            className="abtn xs"
                            onClick={() => setCodesFor(it)}
                            title={t('inv_codes_tip')}
                          >
                            <Icon name="layers" size={13} /> {t('inv_codes_btn')}
                          </button>
                        )}
                        {canManage && (
                          <button className="abtn xs" onClick={() => onAdd(it.productId)}>
                            <Icon name="upload" size={13} /> {t('add')}
                          </button>
                        )}
                        {canManage && (
                          <button
                            className="abtn xs"
                            onClick={() => exportCodes(it.productId, it.title)}
                            disabled={exportingId === it.productId}
                            title={t('inv_export_tip')}
                          >
                            <Icon name="download" size={13} /> {exportingId === it.productId ? '…' : t('export')}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        {rows.length === 0 && (
          <div className="lsrow faint" style={{ fontSize: 12.5, justifyContent: 'center' }}>
            {debouncedSearch.trim() ? t('inv_no_match') : t('inv_no_low')}
          </div>
        )}
        <Pagination
          page={meta?.page ?? 1}
          pages={meta?.pages ?? 1}
          total={meta?.total ?? rows.length}
          shown={rows.length}
          limit={meta?.limit}
          label={t('pg_products')}
          onPage={setPage}
        />
      </div>
      {codesFor && <CodesModal product={codesFor} onClose={() => setCodesFor(null)} />}
    </div>
  )
}

function UploadTab({ stats, defaultProduct }: { stats: InventoryStats[]; defaultProduct?: string }) {
  const { t } = useTranslation()
  const upload = useUploadCodes()
  const { data: historyRes } = useUploadHistory()
  const history = historyRes?.data ?? []
  const fileRef = useRef<HTMLInputElement>(null)
  const [productId, setProductId] = useState(defaultProduct || stats[0]?.productId || '')
  // Follow the product chosen from a Code-stock row's "Add" (defaultProduct
  // changes on each click); useState alone only reads the first value.
  useEffect(() => {
    if (defaultProduct) setProductId(defaultProduct)
  }, [defaultProduct])
  const [fileName, setFileName] = useState('')
  const [items, setItems] = useState<UploadItem[]>([])
  const [invalid, setInvalid] = useState(0)
  const [manualText, setManualText] = useState('')
  const [result, setResult] = useState<{ inserted: number; duplicates: number; invalid: number } | null>(null)

  // Manually-typed codes reuse the same parser + upload mutation as the file path.
  const manualParsed = useMemo(() => parseCodesFile(manualText), [manualText])

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

  const commitManual = () => {
    if (!productId || manualParsed.items.length === 0) return
    upload.mutate(
      { productId, codes: manualParsed.items },
      {
        onSuccess: (res) => {
          setResult(res.data ?? null)
          setManualText('')
        },
      }
    )
  }

  return (
    <div className="formgrid">
      <div className="fieldset">
        <div className="acard pad">
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('inv_bulk_upload_codes')}</h3>
          <label className="alabel">{t('col_product')}</label>
          <select
            className="select"
            style={{ width: '100%', marginBottom: 16 }}
            value={productId}
            onChange={(e) => setProductId(e.target.value)}
          >
            {stats.length === 0 && <option value="">{t('inv_no_code_products')}</option>}
            {stats.map((it) => (
              <option key={it.productId} value={it.productId}>
                {t('inv_option_available', { title: it.title, n: it.available })}
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
            <div style={{ fontWeight: 800, fontSize: 14, color: 'var(--text)' }}>{t('inv_drop_title')}</div>
            <div className="ahint" style={{ marginTop: 6 }}>
              <Trans i18nKey="inv_line_format_max" components={{ m: <span style={{ fontFamily: 'ui-monospace' }} /> }} />
            </div>
            <button className="abtn sm" style={{ marginTop: 14 }} onClick={() => fileRef.current?.click()}>
              {t('inv_choose_file')}
            </button>
          </div>

          {fileName && (
            <div style={{ marginTop: 16, padding: 14, background: 'var(--surface-2)', borderRadius: 'var(--ar-sm)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                <Icon name="file" size={15} />
                <b style={{ fontSize: 13 }}>{fileName}</b>
                <span className="faint" style={{ fontSize: 12, marginInlineStart: 'auto' }}>
                  {t('inv_rows', { n: items.length })}
                </span>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 10 }}>
                <span className="st st-ok" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {t('inv_valid', { n: validCount })}
                </span>
                <span className="st st-warn" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {t('inv_duplicates', { n: dupes })}
                </span>
                <span className="st st-mute" style={{ fontSize: 11 }}>
                  <i className="d" />
                  {t('inv_bad_format', { n: invalid })}
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
              <Icon name="checkc" size={15} />{' '}
              {t('inv_committed', { inserted: result.inserted, duplicates: result.duplicates })}
            </div>
          )}
          {upload.error && (
            <div style={{ marginTop: 12, color: 'var(--danger)', fontSize: 13, fontWeight: 700 }}>
              {apiErrorText(upload.error, 'inv_upload_failed')}
            </div>
          )}

          <div style={{ display: 'flex', gap: 10, marginTop: 16, justifyContent: 'flex-end' }}>
            <button className="abtn" onClick={() => fileRef.current?.click()}>
              {t('inv_choose_another')}
            </button>
            <button className="abtn primary" disabled={validCount === 0 || upload.isPending} onClick={commit}>
              <Icon name="check" size={15} /> {upload.isPending ? t('inv_committing') : t('inv_commit_n', { n: validCount })}
            </button>
          </div>

          {/* Manual entry — type codes directly instead of uploading a file. */}
          <div style={{ marginTop: 22, paddingTop: 18, borderTop: '1px solid var(--border)' }}>
            <label className="alabel">{t('inv_or_type')}</label>
            <div className="ahint" style={{ marginBottom: 8 }}>
              <Trans i18nKey="inv_line_format" components={{ m: <span style={{ fontFamily: 'ui-monospace' }} /> }} />
            </div>
            <textarea
              className="afield"
              style={{ width: '100%', minHeight: 92, fontFamily: 'ui-monospace', resize: 'vertical' }}
              placeholder={'ABC-123-XYZ\nDEF-456-UVW,1234'}
              value={manualText}
              onChange={(e) => setManualText(e.target.value)}
            />
            <div style={{ display: 'flex', gap: 10, marginTop: 12, alignItems: 'center' }}>
              {manualText.trim() && (
                <span className="faint" style={{ fontSize: 12 }}>
                  {t('inv_valid', { n: manualParsed.items.length })}
                  {manualParsed.invalid > 0 ? ` · ${t('inv_bad_format', { n: manualParsed.invalid })}` : ''}
                </span>
              )}
              <button
                className="abtn primary"
                style={{ marginInlineStart: 'auto' }}
                disabled={manualParsed.items.length === 0 || upload.isPending}
                onClick={commitManual}
              >
                <Icon name="check" size={15} />{' '}
                {upload.isPending
                  ? t('inv_committing')
                  : manualParsed.items.length
                    ? t('inv_add_codes_n', { count: manualParsed.items.length })
                    : t('inv_add_codes')}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div className="fieldset">
        <div className="acard">
          <div className="panelhead">
            <Icon name="clock" size={17} />
            <h3>{t('inv_upload_history')}</h3>
          </div>
          <div>
            {history.length === 0 && (
              <div className="lsrow faint" style={{ fontSize: 12.5 }}>
                {t('inv_no_uploads')}
              </div>
            )}
            {history.map((u) => (
              <div className="lsrow" key={u.id} style={{ flexDirection: 'column', alignItems: 'stretch', gap: 6 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <b style={{ fontSize: 13 }}>{u.productTitle || u.productId}</b>
                  <span className="num faint" style={{ marginInlineStart: 'auto', fontSize: 12 }}>
                    {formatDateTime(u.createdAt)}
                  </span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
                  <span className="st st-ok" style={{ fontSize: 10.5 }}>
                    {t('inv_plus_codes', { n: u.inserted })}
                  </span>
                  {u.duplicates > 0 && (
                    <span className="st st-warn" style={{ fontSize: 10.5 }}>
                      {t('inv_dupes', { n: u.duplicates })}
                    </span>
                  )}
                  <span className="faint" style={{ marginInlineStart: 'auto' }}>
                    {t('inv_by', { name: u.uploadedBy ? u.uploadedBy.split('@')[0] : '—' })}
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
        <h3>{t('inv_thresholds_title')}</h3>
        <div className="ph-act">
          <span className="faint" style={{ fontSize: 12 }}>
            {t('inv_enter_to_save')}
          </span>
        </div>
      </div>
      <div className="tablewrap">
        <table className="tbl">
          <thead>
            <tr>
              <th>{t('col_product')}</th>
              <th>{t('inv_available_now')}</th>
              <th>{t('inv_alert_threshold')}</th>
              <th>{t('inv_auto_disable')}</th>
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
                        {t('inv_below_threshold')}
                      </span>
                    ) : (
                      <span className="st st-ok">
                        <i className="d" />
                        {t('inv_level_ok')}
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
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const { data, isFetching } = useLookupCode(code)
  const audit = data?.data

  return (
    <div className="formgrid">
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('inv_lookup_title')}</h3>
        <div className="fsearch" style={{ minWidth: 0, marginBottom: 18 }}>
          <Icon name="search" size={16} />
          <input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={t('inv_lookup_ph')}
            style={{ fontFamily: 'ui-monospace' }}
          />
        </div>
        {isFetching && <LoadingSpinner />}
        {!isFetching && code.trim().length >= 3 && !audit && (
          <EmptyState icon="search" title={t('inv_no_match_code')} sub={t('inv_no_match_code_sub')} />
        )}
        {audit && (
          <>
            <div className="avault" style={{ marginBottom: 18 }}>
              <span className="vc">{audit.code.code}</span>
              <span className="st st-mute">
                <i className="d" />
                {t(`inv_status_${audit.code.status}`, { defaultValue: audit.code.status })}
              </span>
            </div>
            <div className="deflist">
              <div className="defrow">
                <span className="dk">{t('col_product')}</span>
                <span className="dv">{audit.product?.title ?? audit.code.productId}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('inv_code_id')}</span>
                <span className="dv mono">{audit.code.id.slice(-10)}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('status')}</span>
                <span className="dv" style={{ textTransform: 'capitalize' }}>
                  {t(`inv_status_${audit.code.status}`, { defaultValue: audit.code.status })}
                </span>
              </div>
              <div className="defrow">
                <span className="dk">{t('inv_consumed_by')}</span>
                <span className="dv mono">{audit.code.orderId ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('inv_f_delivered_to')}</span>
                <span className="dv">{audit.code.deliveredTo ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('inv_f_delivered_at')}</span>
                <span className="dv">{audit.code.deliveredAt ?? '—'}</span>
              </div>
              <div className="defrow">
                <span className="dk">{t('inv_uploaded_batch')}</span>
                <span className="dv">{audit.code.batch ?? '—'}</span>
              </div>
            </div>
          </>
        )}
      </div>
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 12 }}>{t('inv_audit_trail')}</h3>
        {!audit ? (
          <div className="faint" style={{ fontSize: 13 }}>
            {t('inv_lookup_hint')}
          </div>
        ) : (
          <div className="timeline">
            {(
              [
                ['done', t('inv_tl_uploaded'), audit.code.batch ?? t('inv_tl_batch'), 'checkc'],
                [
                  audit.code.orderId ? 'done' : 'todo',
                  t('inv_tl_reserved'),
                  audit.code.orderId ?? '—',
                  'clock',
                ],
                [
                  audit.code.status === 'delivered' ? 'done' : 'todo',
                  t('inv_tl_delivered'),
                  audit.code.deliveredAt ?? '—',
                  'send',
                ],
                [audit.code.status === 'delivered' ? 'active' : 'todo', t('inv_tl_vault'), '—', 'shield'],
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
