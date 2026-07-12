import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Stepper } from './Stepper'

describe('Stepper', () => {
  it('commits a typed value on blur', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={1} set={set} />)

    const input = screen.getByLabelText('Quantity')
    await user.clear(input)
    await user.type(input, '12')
    await user.tab() // blur

    expect(set).toHaveBeenCalledWith(12)
  })

  it('commits on Enter', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={1} set={set} />)

    const input = screen.getByLabelText('Quantity')
    await user.clear(input)
    await user.type(input, '7{Enter}')

    expect(set).toHaveBeenCalledWith(7)
  })

  it('clamps a typed value above max down to max', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={1} set={set} max={99} />)

    const input = screen.getByLabelText('Quantity')
    await user.clear(input)
    await user.type(input, '500')
    await user.tab()

    expect(set).toHaveBeenCalledWith(99)
  })

  it('clamps a typed 0 up to min', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={5} set={set} min={1} />)

    const input = screen.getByLabelText('Quantity')
    await user.clear(input)
    await user.type(input, '0')
    await user.tab()

    expect(set).toHaveBeenCalledWith(1)
  })

  it('reverts to the current value when left empty (no update)', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={3} set={set} />)

    const input = screen.getByLabelText('Quantity') as HTMLInputElement
    await user.clear(input)
    await user.tab()

    expect(set).not.toHaveBeenCalled()
    expect(input.value).toBe('3')
  })

  it('strips non-digit characters while typing', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    render(<Stepper value={1} set={set} />)

    const input = screen.getByLabelText('Quantity')
    await user.clear(input)
    await user.type(input, '1a2b')
    await user.tab()

    expect(set).toHaveBeenCalledWith(12)
  })

  it('still steps up and down with the +/- buttons, clamped', async () => {
    const user = userEvent.setup()
    const set = vi.fn()
    const { rerender } = render(<Stepper value={5} set={set} />)

    const [minus, plus] = screen.getAllByRole('button')
    await user.click(plus)
    expect(set).toHaveBeenLastCalledWith(6)
    await user.click(minus)
    expect(set).toHaveBeenLastCalledWith(4)

    // clamps at the bounds
    rerender(<Stepper value={99} set={set} max={99} />)
    await user.click(screen.getAllByRole('button')[1])
    expect(set).toHaveBeenLastCalledWith(99)

    rerender(<Stepper value={1} set={set} min={1} />)
    await user.click(screen.getAllByRole('button')[0])
    expect(set).toHaveBeenLastCalledWith(1)
  })
})
