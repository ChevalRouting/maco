import { Suspense, lazy } from 'react'
import { PageHeader, Spinner, Tabs } from 'cheval-ui'
import { ApiKeys } from '../components/ApiKeys'
import { useTabParam } from '../hooks/useTabParam'

const ApiDocsPanel = lazy(() => import('../components/ApiDocsPanel'))

type ApiTab = 'keys' | 'docs'

const tabs: { key: ApiTab; label: string }[] = [
  { key: 'keys', label: 'API Keys' },
  { key: 'docs', label: 'Docs' },
]

export default function Api() {
  const [tab, setTab] = useTabParam<ApiTab>(['keys', 'docs'], 'keys')

  return (
    <div className="space-y-6">
      <PageHeader title="API" description="Manage API keys and explore the maco HTTP API." />
      <Tabs id="api" label="API" tabs={tabs} active={tab} onChange={setTab} />

      <div role="tabpanel" id="api-panel-keys" aria-labelledby="api-tab-keys" hidden={tab !== 'keys'}>
        <ApiKeys />
      </div>

      <div role="tabpanel" id="api-panel-docs" aria-labelledby="api-tab-docs" hidden={tab !== 'docs'}>
        {tab === 'docs' && (
          <Suspense fallback={<Spinner />}>
            <ApiDocsPanel />
          </Suspense>
        )}
      </div>
    </div>
  )
}
