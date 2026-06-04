import { Icon } from './Icon'

interface CheckboxProps {
  on: boolean
  onClick?: () => void
}

/** Square checkbox used in dense table rows (the prototype's `Ck`). */
export function Checkbox({ on, onClick }: CheckboxProps) {
  return (
    <div
      className={'ck-box' + (on ? ' on' : '')}
      onClick={(e) => {
        e.stopPropagation()
        onClick?.()
      }}
    >
      <Icon name="check" size={12} stroke={3} />
    </div>
  )
}
