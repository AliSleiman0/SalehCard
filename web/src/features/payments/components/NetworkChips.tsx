// NetworkChips lets the user pick the USDT network when more than one is offered.
export function NetworkChips({
  networks,
  value,
  onChange,
}: {
  networks: string[]
  value: string
  onChange: (n: string) => void
}) {
  if (networks.length <= 1) return null
  return (
    <div className="row wrap-gap" style={{ gap: 8 }}>
      {networks.map((n) => (
        <span
          key={n}
          className={'badge clickable ' + (n === value ? 'badge-instant' : 'badge-soft')}
          onClick={() => onChange(n)}
          style={{ padding: '6px 12px', textTransform: 'uppercase' }}
        >
          {n}
        </span>
      ))}
    </div>
  )
}
