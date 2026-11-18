import { PageHeader } from 'maco-ui'
import { CertificateSettings } from '../components/CertificateSettings'

export default function Certificates() {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader title="Certificates" />
      <CertificateSettings />
    </div>
  )
}
