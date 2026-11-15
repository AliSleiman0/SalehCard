import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PageHead, Tabs } from '@/components'
import { useCan } from '@/stores/auth'
import { KnowledgeTab } from '../components/KnowledgeTab'
import { ConversationsTab } from '../components/ConversationsTab'

type Tab = 'knowledge' | 'logs'

/** The Help Center's AI assistant. "AI knowledge" is the plain text it answers
 *  from (2026-11-15: replaced the predefined quick questions and the per-article
 *  knowledge base — one box is simpler for admins); "Conversations" shows what
 *  customers asked (gaps + reported answers). Switched on in Settings. */
export default function ChatbotPage() {
  const { t } = useTranslation()
  const can = useCan()
  const [tab, setTab] = useState<Tab>('knowledge')

  return (
    <div className="page">
      <PageHead
        back="/"
        title={t('nav_chatbot')}
        sub="The Help Center's AI assistant: the knowledge it answers customers from, and what they asked. Turn it on or off in Settings."
      />
      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'knowledge', label: 'AI knowledge' },
          { k: 'logs', label: 'Conversations' },
        ]}
      />
      {tab === 'knowledge' && <KnowledgeTab canManage={can('chatbot.manage')} />}
      {tab === 'logs' && <ConversationsTab />}
    </div>
  )
}
