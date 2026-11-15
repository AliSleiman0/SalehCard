import { describe, it, expect } from 'vitest'
import { askFailure, askHistory, askLang, whatsappLink } from './chat'

describe('chat helpers', () => {
  it('builds a wa.me link from a free-typed number', () => {
    expect(whatsappLink('+961 70 123 456')).toBe('https://wa.me/96170123456')
    expect(whatsappLink('0096170123456')).toBe('https://wa.me/96170123456')
    expect(whatsappLink('x')).toBeNull()
  })

  it('asks in Arabic only for the Arabic locale', () => {
    expect(askLang('ar')).toBe('ar')
    expect(askLang('en')).toBe('en')
    expect(askLang('tr')).toBe('en')
  })

  it('keeps the last turns, oldest first, as user/assistant', () => {
    const msgs = Array.from({ length: 9 }, (_, i) => ({
      from: (i % 2 === 0 ? 'user' : 'bot') as 'user' | 'bot',
      text: `t${i}`,
    }))
    const h = askHistory(msgs)
    expect(h).toHaveLength(6)
    expect(h[0]).toEqual({ role: 'assistant', content: 't3' })
    expect(h[5]).toEqual({ role: 'user', content: 't8' })
  })

  it('maps only the AI limit code to "limited"', () => {
    expect(askFailure('CHATBOT_AI_LIMIT')).toBe('limited')
    expect(askFailure('CHATBOT_AI_DISABLED')).toBe('unavailable')
    expect(askFailure(undefined)).toBe('unavailable')
  })
})
