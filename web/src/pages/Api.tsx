import { Suspense, lazy } from 'react'
import { PageHeader, SectionNav, Spinner } from 'maco-ui'
import { ApiKeys } from '../components/ApiKeys'
import { useHasParam } from '../hooks/useHasParam'
import { useTabParam } from '../hooks/useTabParam'

const ApiDocsPanel = lazy(() => import('../components/ApiDocsPanel'))
const TofuReference = lazy(() => import('../components/TofuReference'))
const TofuExamples = lazy(() => import('../components/TofuExamples'))

type ApiSection = 'keys' | 'docs' | 'tofu' | 'tofu-examples'

interface ApiSectionItem {
  key: ApiSection
  label: string
  title?: string
  group: string
}

const sections: ApiSectionItem[] = [
  { key: 'keys', label: 'Keys', title: 'API Keys', group: 'Access' },
  { key: 'docs', label: 'REST API', title: 'REST Reference', group: 'REST' },
  { key: 'tofu', label: 'Reference', title: 'OpenTofu Reference', group: 'OpenTofu' },
  { key: 'tofu-examples', label: 'Examples', title: 'OpenTofu Examples', group: 'OpenTofu' },
]

export default function Api() {
  const [section, select] = useTabParam<ApiSection>(sections.map((item) => item.key), 'keys')
  const linked = useHasParam('tab')

  return (
    <div className="space-y-6">
      <PageHeader title="API" />
      <SectionNav items={sections} active={section} onChange={select} initiallyEntered={linked}>
        {section === 'keys' && (
          <div className="mx-auto max-w-3xl">
            <ApiKeys />
          </div>
        )}
        <Suspense fallback={<Spinner />}>
          {section === 'docs' && <ApiDocsPanel />}
          {section === 'tofu' && <TofuReference />}
          {section === 'tofu-examples' && <TofuExamples />}
        </Suspense>
      </SectionNav>
    </div>
  )
}
