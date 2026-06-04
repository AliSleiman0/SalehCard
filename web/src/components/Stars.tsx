export function Stars({ value = 5, size = 14 }: { value?: number; size?: number }) {
  return (
    <div className="row" style={{ gap: 1, color: '#ffb02e' }}>
      {[0, 1, 2, 3, 4].map((i) => (
        <svg
          key={i}
          width={size}
          height={size}
          viewBox="0 0 24 24"
          fill={i < Math.round(value) ? '#ffb02e' : 'var(--border-strong)'}
        >
          <path d="M12 3l2.6 5.6 6.1.7-4.5 4.2 1.2 6L12 16.8 6.6 19.5l1.2-6L3.3 9.3l6.1-.7z" />
        </svg>
      ))}
    </div>
  )
}
