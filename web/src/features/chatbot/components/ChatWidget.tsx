import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { useLocaleStore } from '@/stores/locale'
import { askChatbot, fetchChatbot, reportAnswer } from '../api/chatbot'
import { askFailure, askHistory, askLang, whatsappLink } from '../lib/chat'

type Msg = {
  from: 'bot' | 'user'
  text: string
  pending?: boolean
  /** An AI-generated answer: shows the AI note + Report. */
  ai?: boolean
  logId?: string
  reported?: boolean
}

// ChatWidget is the storefront Help Center: a floating button opening a chat
// panel. When the AI assistant is on, a signed-in customer types a question and
// it is answered from the admin's knowledge (2026-11-15: the predefined quick
// questions are gone); signed-out visitors are asked to sign in. The support
// contacts + complaint link are always at the bottom. Hidden while there is
// nothing to offer (AI off and no contact set).
export function ChatWidget() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const locale = useLocaleStore((s) => s.locale)
  const signedIn = useAuthStore((s) => s.isAuthenticated)
  const [open, setOpen] = useState(false)
  const [msgs, setMsgs] = useState<Msg[]>([])
  const [query, setQuery] = useState('')
  const [asking, setAsking] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  const { data } = useQuery({
    queryKey: ['chatbot'],
    queryFn: fetchChatbot,
    staleTime: 5 * 60 * 1000,
  })
  const ai = !!data?.data?.ai
  const support = data?.data?.support ?? {}

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' })
  }, [msgs, open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  const wa = support.whatsapp ? whatsappLink(support.whatsapp) : null
  const hasContact = !!(wa || support.phone || support.email)
  if (!ai && !hasContact) return null

  const send = async () => {
    const q = query.trim()
    if (!q || asking) return
    const history = askHistory(msgs.filter((m) => !m.pending))
    setMsgs((m) => [...m, { from: 'user', text: q }, { from: 'bot', text: t('chat_thinking'), pending: true }])
    setQuery('')
    setAsking(true)
    let reply: Msg
    try {
      const res = await askChatbot(q, askLang(locale), history)
      if (res.success && res.data) {
        reply = { from: 'bot', text: res.data.answer, ai: res.data.answered, logId: res.data.logId }
      } else {
        reply = {
          from: 'bot',
          text: askFailure(res.error?.code) === 'limited' ? t('chat_ai_limit') : t('chat_ai_unavailable'),
        }
      }
    } catch {
      reply = { from: 'bot', text: t('chat_ai_unavailable') }
    }
    setMsgs((m) => [...m.filter((x) => !x.pending), reply])
    setAsking(false)
  }

  const report = (i: number) => {
    const m = msgs[i]
    if (!m?.logId || m.reported) return
    setMsgs((all) => all.map((x, j) => (j === i ? { ...x, reported: true } : x)))
    // Best-effort: the thank-you stays; a lost report is not worth an error.
    reportAnswer(m.logId).catch(() => {})
  }

  return (
    <>
      {open && (
        <div className="chatpanel" role="dialog" aria-label={t('chat_title')}>
          <div className="chathead">
            <span className="row" style={{ gap: 8, fontWeight: 800 }}>
              <Icon name="chat" size={17} /> {t('chat_title')}
            </span>
            <button className="icon-btn" style={{ width: 34, height: 34 }} onClick={() => setOpen(false)} title="Close">
              <Icon name="close" size={16} />
            </button>
          </div>

          <div className="chatlist" ref={listRef}>
            <div className="chatmsg bot">{ai ? t('chat_greeting_ai') : t('chat_greeting_support')}</div>
            {msgs.map((m, i) => (
              <div key={i} className={`chatmsg ${m.from}`} dir="auto" style={m.pending ? { opacity: 0.6 } : undefined}>
                {m.text}
                {m.ai && (
                  <div className="tiny faint" style={{ marginTop: 6 }}>
                    {t('chat_ai_note')}
                    {' · '}
                    {m.reported ? (
                      t('chat_ai_reported')
                    ) : m.logId ? (
                      <a className="clickable" style={{ textDecoration: 'underline' }} onClick={() => report(i)}>
                        {t('chat_ai_report')}
                      </a>
                    ) : null}
                  </div>
                )}
              </div>
            ))}
          </div>

          <div className="chatfoot">
            {ai &&
              (signedIn ? (
                <form
                  className="row"
                  style={{ gap: 8 }}
                  onSubmit={(e) => {
                    e.preventDefault()
                    void send()
                  }}
                >
                  <input
                    className="field"
                    style={{ padding: '9px 12px', fontSize: 14, flex: 1 }}
                    placeholder={t('chat_ask_ph')}
                    maxLength={500}
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    dir="auto"
                  />
                  <button
                    type="submit"
                    className="btn btn-primary"
                    style={{ padding: '9px 14px' }}
                    disabled={asking || !query.trim()}
                    aria-label={t('chat_send')}
                    title={t('chat_send')}
                  >
                    <Icon name="arrow" size={16} />
                  </button>
                </form>
              ) : (
                <div className="tiny">
                  <a
                    className="clickable"
                    style={{ fontWeight: 700 }}
                    onClick={() => {
                      setOpen(false)
                      navigate('/login')
                    }}
                  >
                    {t('login')}
                  </a>{' '}
                  <span className="faint">{t('chat_sign_in_to_ask')}</span>
                </div>
              ))}
            {hasContact && (
              <div className="row wrap-gap" style={{ gap: 10, marginTop: ai ? 8 : 0, alignItems: 'center' }}>
                <span className="tiny faint">{t('chat_need_help')}</span>
                {wa && (
                  <a className="tiny" style={{ fontWeight: 700 }} href={wa} target="_blank" rel="noreferrer">
                    WhatsApp
                  </a>
                )}
                {support.phone && (
                  <a className="tiny" style={{ fontWeight: 700 }} href={`tel:${support.phone.replace(/\s+/g, '')}`}>
                    {support.phone}
                  </a>
                )}
                {support.email && (
                  <a className="tiny" style={{ fontWeight: 700 }} href={`mailto:${support.email}`}>
                    {support.email}
                  </a>
                )}
              </div>
            )}
            <div className="row" style={{ gap: 10, marginTop: 8, alignItems: 'center' }}>
              {!hasContact && <span className="tiny faint">{t('chat_need_help')}</span>}
              <a
                className="tiny clickable"
                style={{ fontWeight: 700 }}
                onClick={() => {
                  setOpen(false)
                  navigate('/complaints/new')
                }}
              >
                {t('complaint_submit')}
              </a>
            </div>
          </div>
        </div>
      )}
      <button
        className="chatfab"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        aria-label={t('chat_title')}
        title={t('chat_title')}
      >
        <Icon name={open ? 'close' : 'chat'} size={22} />
      </button>
    </>
  )
}
