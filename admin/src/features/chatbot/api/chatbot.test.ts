import { describe, it, expect } from 'vitest'
import { knowledgeChars, paragraphCount } from './chatbot'

describe('knowledgeChars', () => {
  it('counts code points of both texts like the server (trimmed, CRLF as one)', () => {
    expect(knowledgeChars({ general: '  ab\r\ncd  ', androidOnly: 'شحن😀' })).toBe(5 + 4)
  })
})

describe('paragraphCount', () => {
  it('splits on blank lines and ignores empty ones', () => {
    expect(paragraphCount('one\nstill one\n\ntwo\r\n  \r\nthree\n\n\n')).toBe(3)
    expect(paragraphCount('   ')).toBe(0)
  })
})
