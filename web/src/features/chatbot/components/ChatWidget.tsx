import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Icon } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { fetchChatbot, type ChatEntry } from '../api/chatbot'
import { filterEntries, followUps, text, topLevel, whatsappLink } from '../lib/chat'

type Msg = { from: 'bot' | 'user'; text: string }

// ChatWidget is the storefront help bot (client msg #74): a floating button
// opening a chat-style panel of PREDEFINED questions maintained in the admin
// console. Tapping a question shows its answer, then its follow-ups; a filter
// box searches questions + keywords; "Still need help?" links to support.
// Hidden entirely while the admin has not added any questions.
export function ChatWidget() {
  const { t } = useTranslation()
  const locale = useLocaleStore((s) => s.locale)
  const [open, setOpen] = useState(false)
  const [msgs, setMsgs] = useState<Msg[]>([])
  const [current, setCurrent] = useState<string | null>(null) // last answered top-level id
  const [query, setQuery] = useState('')
  const listRef = useRef<HTMLDivElement>(null)

  const { data } = useQuery({
    queryKey: ['chatbot'],
    queryFn: fetchChatbot,
    staleTime: 5 * 60 * 1000,
  })
  const entries = useMemo(() => data?.data?.entries ?? [], [data])
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

  if (entries.length === 0) return null

  const ask = (e: ChatEntry) => {
    setMsgs((m) => [
      ...m,
      { from: 'user', text: text(e.question, locale) },
      { from: 'bot', text: text(e.answer, locale) },
    ])
    setQuery('')
    // After a top-level answer offer its follow-ups; after a follow-up, the
    // remaining follow-ups of the same parent.
    setCurrent(e.parentId ?? e.id)
  }

  const filtered = filterEntries(entries, query)
  const suggestions = query.trim()
    ? filtered
    : current
      ? followUps(entries, current)
      : topLevel(entries)
  const wa = support.whatsapp ? whatsappLink(support.whatsapp) : null

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
            <div className="chatmsg bot">{t('chat_greeting')}</div>
            {msgs.map((m, i) => (
              <div key={i} className={`chatmsg ${m.from}`}>
                {m.text}
              </div>
            ))}
          </div>

          <div className="chatchips">
            {suggestions.map((e) => (
              <button key={e.id} className="chatchip" onClick={() => ask(e)}>
                {text(e.question, locale)}
              </button>
            ))}
            {query.trim() && filtered.length === 0 && <span className="tiny faint">{t('no_results')}</span>}
            {current && !query.trim() && (
              <button className="chatchip ghost" onClick={() => setCurrent(null)}>
                {t('chat_all_questions')}
              </button>
            )}
          </div>

          <div className="chatfoot">
            <input
              className="field"
              style={{ padding: '9px 12px', fontSize: 14 }}
              placeholder={t('chat_filter_ph')}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            {(wa || support.phone || support.email) && (
              <div className="row wrap-gap" style={{ gap: 10, marginTop: 8, alignItems: 'center' }}>
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
