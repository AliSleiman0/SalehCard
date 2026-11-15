import { afterEach, describe, expect, it } from 'vitest'
import i18n from '@/i18n/config'
import { ApiError } from '@/lib/api-client'
import { apiErrorText } from './apiErrorText'

describe('apiErrorText', () => {
  afterEach(() => i18n.changeLanguage('en'))

  it('keeps the server message in English', async () => {
    await i18n.changeLanguage('en')
    expect(apiErrorText(new ApiError(409, 'LAST_ADMIN', 'Server says no'), 'cancel')).toBe('Server says no')
  })

  it('uses the translated code in Arabic when a key exists', async () => {
    i18n.addResource('ar', 'common', 'err_TEST_CODE', 'رسالة مترجمة')
    await i18n.changeLanguage('ar')
    expect(apiErrorText(new ApiError(400, 'TEST_CODE', 'English text'), 'cancel')).toBe('رسالة مترجمة')
  })

  it('falls back to the server message for an unknown code', async () => {
    await i18n.changeLanguage('ar')
    expect(apiErrorText(new ApiError(400, 'NO_SUCH_CODE_X', 'English text'), 'cancel')).toBe('English text')
  })

  it('uses the fallback key for a non-API error', async () => {
    await i18n.changeLanguage('ar')
    expect(apiErrorText(new Error('boom'), 'cancel')).toBe(i18n.t('cancel'))
  })
})
