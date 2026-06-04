export function Tabs({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: string; label: string }[]
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="tabs">
      {tabs.map((tab) => (
        <button
          key={tab.value}
          className={tab.value === value ? 'on' : ''}
          onClick={() => onChange(tab.value)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  )
}
