import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Icon,
  PageHead,
  Chip,
  Pagination,
  Toggle,
  Modal,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import {
  useBridgeDevices,
  useBridgeCommands,
  useCreateDevice,
  useUpdateDevice,
  useRotateToken,
  useDeleteDevice,
  useCheckBalance,
  useRetryCommand,
  useCancelCommand,
} from '../hooks/useBridge'
import type { BridgeCommandStatus, BridgeDevice } from '../api/bridge'

const STATUS_FILTERS: [BridgeCommandStatus | '', string][] = [
  ['', 'All'],
  ['queued', 'Queued'],
  ['leased', 'In progress'],
  ['succeeded', 'Succeeded'],
  ['failed', 'Failed'],
  ['cancelled', 'Cancelled'],
]

const STATUS_CLASS: Record<BridgeCommandStatus, string> = {
  queued: 'st st-warn',
  leased: 'st st-warn',
  succeeded: 'st st-ok',
  failed: 'st st-danger',
  cancelled: 'st st-mute',
}

// A device is "online" if it checked in within the last two minutes.
const isOnline = (d: BridgeDevice) =>
  !!d.lastSeenAt && Date.now() - new Date(d.lastSeenAt).getTime() < 2 * 60_000

export default function BridgePage() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<BridgeCommandStatus | ''>('')
  const [page, setPage] = useState(1)
  const [registering, setRegistering] = useState(false)
  const [revealedToken, setRevealedToken] = useState<string | null>(null)

  const devicesQ = useBridgeDevices()
  const commandsQ = useBridgeCommands({ page, status: status || undefined })
  const createDevice = useCreateDevice()
  const updateDevice = useUpdateDevice()
  const rotateToken = useRotateToken()
  const deleteDevice = useDeleteDevice()
  const checkBalance = useCheckBalance()
  const retry = useRetryCommand()
  const cancel = useCancelCommand()

  const devices = useMemo(() => devicesQ.data?.data ?? [], [devicesQ.data])
  const commands = useMemo(() => commandsQ.data?.data ?? [], [commandsQ.data])
  const meta = commandsQ.data?.meta

  const onRegister = (name: string, providers: string[]) => {
    createDevice.mutate(
      { name, providers },
      {
        onSuccess: (res) => {
          setRegistering(false)
          if (res.data?.token) setRevealedToken(res.data.token)
        },
      },
    )
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_operations'), t('nav_bridge')]}
        title={t('nav_bridge')}
        sub={`${devices.length} device${devices.length === 1 ? '' : 's'}`}
      >
        <button className="abtn primary" onClick={() => setRegistering(true)}>
          <Icon name="plus" size={15} /> Register device
        </button>
      </PageHead>

      {/* Devices */}
      <div className="acard">
        <div className="panelhead">
          <Icon name="box" size={17} />
          <h3>Devices</h3>
        </div>
        {devicesQ.isLoading ? (
          <LoadingSpinner />
        ) : devicesQ.isError ? (
          <ErrorState message="Couldn't load devices." onRetry={() => devicesQ.refetch()} />
        ) : devices.length === 0 ? (
          <EmptyState
            title="No bridge devices"
            sub="Register the dual-SIM phone that fulfills mobile recharges, then enter its token in the bridge app."
          />
        ) : (
          <div>
            {devices.map((d) => (
              <div
                key={d.id}
                style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                  <span className={isOnline(d) ? 'st st-ok' : 'st st-mute'}>
                    <i className="d" />
                    {isOnline(d) ? 'online' : 'offline'}
                  </span>
                  <b style={{ fontSize: 15 }}>{d.name}</b>
                  {d.providers.map((p) => (
                    <span key={p} className="bdg">
                      {p}
                    </span>
                  ))}
                  {d.appVersion && (
                    <span className="faint" style={{ fontSize: 12 }}>
                      v{d.appVersion}
                    </span>
                  )}
                  <div
                    style={{ marginInlineStart: 'auto', display: 'flex', alignItems: 'center', gap: 12 }}
                  >
                    <Toggle
                      on={d.enabled}
                      onClick={() => updateDevice.mutate({ id: d.id, patch: { enabled: !d.enabled } })}
                    />
                    <button className="abtn xs" onClick={() => checkBalance.mutate(d.id)}>
                      Check balance
                    </button>
                    <button
                      className="abtn xs"
                      onClick={() =>
                        rotateToken.mutate(d.id, {
                          onSuccess: (res) => res.data?.token && setRevealedToken(res.data.token),
                        })
                      }
                    >
                      Rotate token
                    </button>
                    <span
                      className="iact danger"
                      title="Remove device"
                      onClick={() => {
                        if (confirm(`Remove ${d.name}? Its token stops working immediately.`)) {
                          deleteDevice.mutate(d.id)
                        }
                      }}
                    >
                      <Icon name="trash" size={15} />
                    </span>
                  </div>
                </div>
                <div className="faint" style={{ fontSize: 12.5, marginTop: 8, display: 'flex', gap: 16, flexWrap: 'wrap' }}>
                  {d.touchBalance != null && (
                    <span>Touch ${d.touchBalance.toFixed(2)} {d.touchValidity && `(exp ${d.touchValidity})`}</span>
                  )}
                  {d.alfaBalance != null && (
                    <span>Alfa ${d.alfaBalance.toFixed(2)} {d.alfaValidity && `(exp ${d.alfaValidity})`}</span>
                  )}
                  {d.lastSeenAt && <span>· last seen {new Date(d.lastSeenAt).toLocaleString()}</span>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Commands */}
      <div className="acard" style={{ marginTop: 18 }}>
        <div className="panelhead">
          <Icon name="bag" size={17} />
          <h3>Recharge commands</h3>
        </div>
        <div className="toolbar" style={{ gap: 16, flexWrap: 'wrap' }}>
          <div className="chiprow">
            {STATUS_FILTERS.map(([k, l]) => (
              <Chip
                key={k || 'all'}
                on={status === k}
                onClick={() => {
                  setStatus(k)
                  setPage(1)
                }}
              >
                {l}
              </Chip>
            ))}
          </div>
        </div>

        {commandsQ.isLoading ? (
          <LoadingSpinner />
        ) : commandsQ.isError ? (
          <ErrorState message="Couldn't load commands." onRetry={() => commandsQ.refetch()} />
        ) : commands.length === 0 ? (
          <EmptyState title="No commands" sub="Recharge orders enqueue commands here for the bridge device." />
        ) : (
          <>
            <div>
              {commands.map((c) => (
                <div key={c.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    <span className="bdg">{c.provider}</span>
                    <b style={{ fontSize: 14 }}>{c.type}</b>
                    {c.recipientNumber && <span className="faint" style={{ fontSize: 13 }}>→ {c.recipientNumber}</span>}
                    {c.amount != null && <span style={{ fontSize: 13 }}>${c.amount.toFixed(2)}</span>}
                    {c.cardCodeMasked && <span className="faint" style={{ fontSize: 12.5 }}>{c.cardCodeMasked}</span>}
                    <span style={{ marginInlineStart: 'auto', display: 'flex', gap: 10, alignItems: 'center' }}>
                      <span className={STATUS_CLASS[c.status]}>
                        <i className="d" />
                        {c.status}
                      </span>
                      {(c.status === 'failed' || c.status === 'cancelled') && (
                        <button className="abtn xs" onClick={() => retry.mutate(c.id)}>
                          Retry
                        </button>
                      )}
                      {(c.status === 'queued' || c.status === 'leased') && (
                        <button className="abtn xs" onClick={() => cancel.mutate(c.id)}>
                          Cancel
                        </button>
                      )}
                    </span>
                  </div>
                  <div className="faint" style={{ fontSize: 12.5, marginTop: 8, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
                    <span>attempt {c.attempts}</span>
                    {c.statusCode != null && <span>· code {c.statusCode}</span>}
                    {c.orderId && <span>· order …{c.orderId.slice(-6)}</span>}
                    {c.failReason && <span style={{ color: 'var(--danger)' }}>· {c.failReason}</span>}
                    {c.rawReply && <span title={c.rawReply}>· reply: {c.rawReply.slice(0, 48)}</span>}
                    <span>· {new Date(c.createdAt).toLocaleString()}</span>
                  </div>
                </div>
              ))}
            </div>
            <Pagination
              page={meta?.page ?? 1}
              pages={meta?.pages ?? 1}
              total={meta?.total ?? commands.length}
              shown={commands.length}
              limit={meta?.limit}
              label={t('nav_bridge')}
              onPage={setPage}
            />
          </>
        )}
      </div>

      {registering && (
        <RegisterModal
          onClose={() => setRegistering(false)}
          onSubmit={onRegister}
          pending={createDevice.isPending}
        />
      )}
      {revealedToken && <TokenModal token={revealedToken} onClose={() => setRevealedToken(null)} />}
    </div>
  )
}

function RegisterModal({
  onClose,
  onSubmit,
  pending,
}: {
  onClose: () => void
  onSubmit: (name: string, providers: string[]) => void
  pending: boolean
}) {
  const [name, setName] = useState('')
  const [touch, setTouch] = useState(true)
  const [alfa, setAlfa] = useState(true)
  const providers = [...(touch ? ['touch'] : []), ...(alfa ? ['alfa'] : [])]
  return (
    <Modal onClose={onClose} maxWidth={420}>
      <div style={{ padding: 20 }}>
        <h3 style={{ marginBottom: 12 }}>Register bridge device</h3>
        <label className="alabel">Name</label>
        <input
          className="afield"
          value={name}
          placeholder="Shop phone 1"
          onChange={(e) => setName(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          Operators
        </label>
        <label style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 6 }}>
          <input type="checkbox" checked={touch} onChange={(e) => setTouch(e.target.checked)} /> MTC Touch
        </label>
        <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input type="checkbox" checked={alfa} onChange={(e) => setAlfa(e.target.checked)} /> Alfa
        </label>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 18 }}>
          <button className="abtn" onClick={onClose}>
            Cancel
          </button>
          <button
            className="abtn primary"
            disabled={pending || !name.trim() || providers.length === 0}
            onClick={() => onSubmit(name.trim(), providers)}
          >
            Register
          </button>
        </div>
      </div>
    </Modal>
  )
}

function TokenModal({ token, onClose }: { token: string; onClose: () => void }) {
  return (
    <Modal onClose={onClose} maxWidth={480}>
      <div style={{ padding: 20 }}>
        <h3 style={{ marginBottom: 8 }}>Device token</h3>
        <div className="ahint" style={{ margin: '0 0 12px' }}>
          Copy this now — it's shown <b>once</b> and can't be retrieved later. Enter it in the bridge
          app on the device. To replace it, rotate the token.
        </div>
        <code
          style={{
            display: 'block',
            padding: 12,
            background: 'var(--surface-2)',
            borderRadius: 'var(--r-md)',
            wordBreak: 'break-all',
            fontSize: 13,
          }}
        >
          {token}
        </code>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 16 }}>
          <button className="abtn" onClick={() => navigator.clipboard?.writeText(token)}>
            Copy
          </button>
          <button className="abtn primary" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </Modal>
  )
}
