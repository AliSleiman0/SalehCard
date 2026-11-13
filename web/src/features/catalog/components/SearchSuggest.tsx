import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Icon } from '@/components'
import { useDebouncedValue } from '@/lib/useDebouncedValue'
import { useLocaleStore } from '@/stores/locale'
import { useProducts } from '../hooks/useProducts'
import { buildSuggestions, highlight } from '../lib/suggest'

/**
 * Header search with live suggestions (client feedback #11/#86): as soon as the
 * customer types, a dropdown lists products whose name contains those letters,
 * prefix matches first then A to Z. Arrow keys + Enter pick a product; Enter
 * with nothing highlighted opens the full results page.
 */
export function SearchSuggest() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const locale = useLocaleStore((s) => s.locale)
  const [text, setText] = useState('')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const wrap = useRef<HTMLDivElement>(null)
  const term = useDebouncedValue(text.trim(), 180)

  // Candidates come back A to Z from the server so prefix matches are not cut
  // off by an arbitrary first page; buildSuggestions then ranks prefix first.
  const query = useProducts({ q: term, limit: 60, sort: 'name' }, { enabled: term.length > 0 })
  const items = term ? buildSuggestions(query.data?.data ?? [], term, locale) : []
  const showList = open && term.length > 0

  // Close when clicking anywhere outside the search box.
  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [])

  const go = (path: string) => {
    setOpen(false)
    setActive(-1)
    setText('')
    navigate(path)
  }
  const submit = () => {
    if (active >= 0 && items[active]) return go('/product/' + items[active].id)
    if (text.trim()) go('/search?q=' + encodeURIComponent(text.trim()))
  }

  return (
    <div className="searchbar desktop-only suggest-wrap" ref={wrap}>
      <Icon name="search" size={18} />
      <input
        placeholder={t('search_ph')}
        value={text}
        role="combobox"
        aria-expanded={showList}
        aria-controls="search-suggest-list"
        aria-autocomplete="list"
        onChange={(e) => {
          setText(e.target.value)
          setOpen(true)
          setActive(-1)
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={(e) => {
          if (e.key === 'ArrowDown') {
            e.preventDefault()
            setActive((a) => Math.min(items.length - 1, a + 1))
          } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setActive((a) => Math.max(-1, a - 1))
          } else if (e.key === 'Enter') {
            submit()
          } else if (e.key === 'Escape') {
            setOpen(false)
          }
        }}
      />
      {showList && (
        <div className="suggest-list" id="search-suggest-list" role="listbox">
          {items.map((s, i) => {
            const [a, b, c] = highlight(s.name, term)
            return (
              <div
                key={s.id}
                role="option"
                aria-selected={i === active}
                className={'suggest-row' + (i === active ? ' on' : '')}
                onMouseEnter={() => setActive(i)}
                onMouseDown={(e) => {
                  e.preventDefault() // keep focus; navigate on click
                  go('/product/' + s.id)
                }}
              >
                {s.image ? <img src={s.image} alt="" className="suggest-img" /> : <Icon name="search" size={15} />}
                <span className="suggest-name">
                  {a}
                  <b>{b}</b>
                  {c}
                </span>
              </div>
            )
          })}
          {items.length === 0 && (
            <div className="suggest-empty">{query.isFetching ? t('loading', { defaultValue: 'Searching…' }) : t('no_results')}</div>
          )}
          {items.length > 0 && (
            <div
              className="suggest-row suggest-all"
              onMouseDown={(e) => {
                e.preventDefault()
                go('/search?q=' + encodeURIComponent(text.trim()))
              }}
            >
              <Icon name="search" size={15} />
              <span>
                {t('search_all', { defaultValue: 'See all results for' })} “{text.trim()}”
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
