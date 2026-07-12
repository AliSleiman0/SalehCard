import { QRCodeSVG } from 'qrcode.react'

// QrCode renders the deposit address as a scannable SVG on a white card (so it
// scans in dark mode too). The deposit page is route-lazy, so qrcode.react never
// touches the initial bundle.
export function QrCode({ value, size = 180 }: { value: string; size?: number }) {
  return (
    <div style={{ background: '#fff', padding: 14, borderRadius: 14, lineHeight: 0 }}>
      <QRCodeSVG value={value} size={size} level="M" />
    </div>
  )
}
