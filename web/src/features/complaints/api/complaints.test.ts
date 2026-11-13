import { describe, it, expect } from 'vitest'
import { statusBadgeClass, validateComplaint } from './complaints'

describe('validateComplaint', () => {
  it('requires a subject and a message', () => {
    expect(validateComplaint({ subject: ' ', message: 'x' })).toBe('complaint_err_subject')
    expect(validateComplaint({ subject: 'x', message: '' })).toBe('complaint_err_message')
    expect(validateComplaint({ subject: 'Late code', message: 'It never arrived' })).toBe('')
  })
  it('enforces the server limits', () => {
    expect(validateComplaint({ subject: 'x'.repeat(121), message: 'm' })).toBe('complaint_err_subject_long')
    expect(validateComplaint({ subject: 's', message: 'x'.repeat(2001) })).toBe('complaint_err_message_long')
  })
})

describe('statusBadgeClass', () => {
  it('maps every status', () => {
    expect(statusBadgeClass('open')).toContain('badge-agent')
    expect(statusBadgeClass('resolved')).toContain('badge-instant')
    expect(statusBadgeClass('closed')).toContain('badge-soft')
  })
})
