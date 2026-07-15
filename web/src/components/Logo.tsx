export function Logo({
  size = 30,
  mono = false,
  agent = false,
}: {
  size?: number
  mono?: boolean
  agent?: boolean
}) {
  const s = size
  return (
    <div className="row" style={{ gap: s * 0.34 }}>
      <div
        style={{
          width: s,
          height: s,
          borderRadius: s * 0.3,
          position: 'relative',
          background: mono ? 'currentColor' : 'var(--grad)',
          boxShadow: mono ? 'none' : '0 6px 18px -6px rgba(138,59,255,.8)',
          display: 'grid',
          placeItems: 'center',
          flex: 'none',
        }}
      >
        {/* geometric "S" spark mark */}
        <svg width={s * 0.6} height={s * 0.6} viewBox="0 0 24 24" fill="none">
          <path
            d="M16 5H10a3 3 0 0 0 0 6h4a3 3 0 0 1 0 6H7"
            stroke="#fff"
            strokeWidth="2.6"
            strokeLinecap="round"
          />
          <circle cx="18.5" cy="6" r="1.7" fill={agent ? '#ffd76b' : '#22e3c8'} />
        </svg>
      </div>
      <div
        className="logo-word"
        style={{
          fontFamily: 'var(--font-display)',
          fontSize: s * 0.62,
          letterSpacing: '-.02em',
          lineHeight: 1,
        }}
      >
        Saleh<span className="grad-text">Card</span>
        {agent && (
          <span
            className="badge badge-agent"
            style={{ marginInlineStart: 8, verticalAlign: 'middle' }}
          >
            AGENT
          </span>
        )}
      </div>
    </div>
  )
}
