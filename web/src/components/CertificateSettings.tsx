import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { EntryRow, NoticeBanner, PreferencesGroup, Row, Segmented, StatusChip, TagInput, Textarea, type StatusIntent } from 'maco-ui'
import { toast } from 'sonner'
import { errorMessage, getTLS, updateTLS, type TLSStatus, type TLSUpdate, type TLSView } from '../api'
import { useResource } from '../hooks/useResource'
import { useSession } from '../hooks/useSession'
import { FormActions } from './FormActions'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'

type Mode = 'self-signed' | 'file' | 'acme'

interface CertificateDraft {
  mode: Mode
  certPEM: string
  keyPEM: string
  directory: string
  email: string
  domains: string[]
  httpAddr: string
  caPEM: string
  eabKID: string
  eabKey: string
}

interface PemFieldProps {
  id: string
  label: string
  value: string
  placeholder: string
  help?: ReactNode
  onChange: (value: string) => void
}

interface ModeOption {
  value: Mode
  label: string
}

interface CertificateChip {
  intent: StatusIntent
  label: string
}

interface CertificateRowProps {
  status: TLSStatus
  acme: boolean
}

const MODE_OPTIONS: ModeOption[] = [
  { value: 'self-signed', label: 'Self-Signed' },
  { value: 'file', label: 'Custom' },
  { value: 'acme', label: 'ACME' },
]

const LETS_ENCRYPT = 'https://acme-v02.api.letsencrypt.org/directory'
const DAY = 24 * 60 * 60 * 1000
const domainPattern = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]{2,63}\.?$/i

function draftFrom(view: TLSView): CertificateDraft {
  return {
    mode: view.mode as Mode,
    certPEM: view.cert_pem,
    keyPEM: '',
    directory: view.acme.directory,
    email: view.acme.email,
    domains: view.acme.domains ?? [],
    httpAddr: view.acme.http_addr,
    caPEM: view.acme.ca_pem,
    eabKID: view.acme.eab_kid,
    eabKey: '',
  }
}

function updateFrom(draft: CertificateDraft, view: TLSView): TLSUpdate {
  const keepFiles = draft.mode === 'file' && view.mode === 'file' && draft.keyPEM === '' && draft.certPEM === view.cert_pem
  return {
    mode: draft.mode,
    cert_pem: keepFiles ? '' : draft.certPEM,
    key_pem: keepFiles ? '' : draft.keyPEM,
    acme: {
      directory: draft.directory,
      email: draft.email,
      domains: draft.domains,
      http_addr: draft.httpAddr,
      ca_pem: draft.caPEM,
      eab_kid: draft.eabKID,
      eab_key: draft.eabKey,
    },
  }
}

function validDomain(value: string) {
  return domainPattern.test(value) ? null : 'Use a fully qualified name'
}

function statusIntent(status: TLSStatus): CertificateChip {
  if (!status.ready && status.error) return { intent: 'danger', label: 'Failed' }
  if (!status.ready) return { intent: 'neutral', label: 'Pending' }
  const remaining = new Date(status.not_after).getTime() - Date.now()
  if (remaining <= 0) return { intent: 'danger', label: 'Expired' }
  if (remaining < 14 * DAY) return { intent: 'warning', label: 'Expiring' }
  return { intent: 'success', label: 'Valid' }
}

function sourceLabel(source: string) {
  return MODE_OPTIONS.find((option) => option.value === source)?.label ?? source
}

function PemField({ id, label, value, placeholder, help, onChange }: PemFieldProps) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
      <label htmlFor={id} className="text-sm font-medium">{label}</label>
      <Textarea
        id={id}
        value={value}
        rows={5}
        spellCheck={false}
        placeholder={placeholder}
        className="font-mono text-xs"
        onChange={(event) => onChange(event.target.value)}
      />
      {help && <p className="text-xs text-muted-foreground">{help}</p>}
    </div>
  )
}

function CertificateRow({ status, acme }: CertificateRowProps) {
  const chip = statusIntent(status)
  const detail = status.ready ? `${status.issuer} · expires ${new Date(status.not_after).toLocaleDateString()}` : (status.error || 'ACME issuance in progress')
  return (
    <Row title={status.name !== 'fallback' ? status.name : acme ? 'Fallback' : 'Default'} subtitle={`${sourceLabel(status.source)} · ${detail}`}>
      <StatusChip intent={chip.intent} label={chip.label} />
    </Row>
  )
}

export function CertificateSettings() {
  const tls = useResource(getTLS, null, 'tls')
  const { admin } = useSession()
  const [view, setView] = useState<TLSView | null>(null)
  const [draft, setDraft] = useState<CertificateDraft | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const shown = useRef<TLSView | null>(null)

  useEffect(() => {
    const next = tls.data
    if (!next) return
    const previous = shown.current
    shown.current = next
    setView(next)
    setDraft((current) => (!current || !previous || JSON.stringify(current) === JSON.stringify(draftFrom(previous)) ? draftFrom(next) : current))
  }, [tls.data])

  const pending = view?.mode === 'acme'
  const refresh = tls.refresh
  useEffect(() => {
    if (!pending) return
    const timer = window.setInterval(refresh, 5000)
    return () => window.clearInterval(timer)
  }, [pending, refresh])

  if (!view || !draft) return <ResourceNotice resource={tls} name="certificates" />

  const dirty = JSON.stringify(draft) !== JSON.stringify(draftFrom(view))
  const editable = admin && !view.locked
  const keepingFiles = draft.mode === 'file' && view.mode === 'file'
  const fileIncomplete = draft.mode === 'file' && !keepingFiles && (!draft.certPEM.trim() || !draft.keyPEM.trim())
  const acmeIncomplete = draft.mode === 'acme' && draft.domains.length === 0
  const set = (patch: Partial<CertificateDraft>) => setDraft({ ...draft, ...patch })

  async function save(event: FormEvent) {
    event.preventDefault()
    if (!view || !draft || busy || !dirty) return
    setBusy(true)
    setError('')
    try {
      const saved = await updateTLS(updateFrom(draft, view))
      shown.current = saved
      setView(saved)
      setDraft(draftFrom(saved))
      toast.success('Certificate settings applied')
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={save} className="space-y-3">
      <JobNotice error={error} />
      {view.locked && <NoticeBanner intent="info">Set by --tls-cert and --tls-key on the command line</NoticeBanner>}
      {view.error && <NoticeBanner intent="danger">{view.error}</NoticeBanner>}
      <fieldset disabled={!editable || busy} className="space-y-3">
        <PreferencesGroup title="Certificate" description="Served by the web UI and API">
          <Row title="Source">
            <Segmented value={draft.mode} onChange={(mode) => set({ mode })} options={MODE_OPTIONS} />
          </Row>
          {draft.mode === 'self-signed' && <Row subtitle="Generated in the data directory; browsers warn until it is trusted" />}
          {draft.mode === 'file' && (
            <>
              <PemField
                id="tls-cert"
                label="Certificate Chain"
                value={draft.certPEM}
                placeholder="-----BEGIN CERTIFICATE-----"
                onChange={(certPEM) => set({ certPEM })}
              />
              <PemField
                id="tls-key"
                label="Private Key"
                value={draft.keyPEM}
                placeholder={keepingFiles ? 'Leave empty to keep the current key' : '-----BEGIN PRIVATE KEY-----'}
                help={keepingFiles ? 'Files changed on disk are reloaded automatically' : undefined}
                onChange={(keyPEM) => set({ keyPEM })}
              />
            </>
          )}
        </PreferencesGroup>

        {draft.mode === 'acme' && (
          <PreferencesGroup title="ACME" description="Certificates are requested on save and renewed 30 days before expiry">
            <EntryRow
              id="tls-acme-directory"
              title="Directory"
              value={draft.directory}
              help={draft.directory === LETS_ENCRYPT ? "Let's Encrypt" : undefined}
              onChange={(event) => set({ directory: event.target.value })}
            />
            <EntryRow id="tls-acme-email" title="Contact Email" type="email" value={draft.email} onChange={(event) => set({ email: event.target.value })} />
            <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
              <span className="text-sm font-medium">Domains</span>
              <TagInput values={draft.domains} mono placeholder="maco.example.com" validate={validDomain} onChange={(domains) => set({ domains })} />
            </div>
            <EntryRow
              id="tls-acme-http"
              title="Challenge Listener"
              value={draft.httpAddr}
              help="http-01 challenges; the ACME server must reach it on port 80"
              onChange={(event) => set({ httpAddr: event.target.value })}
            />
            <PemField
              id="tls-acme-ca"
              label="Directory CA"
              value={draft.caPEM}
              placeholder="Optional PEM bundle for a private ACME server"
              onChange={(caPEM) => set({ caPEM })}
            />
            <EntryRow id="tls-acme-eab-kid" title="EAB Key ID" value={draft.eabKID} onChange={(event) => set({ eabKID: event.target.value })} />
            <EntryRow
              id="tls-acme-eab-key"
              title="EAB HMAC Key"
              type="password"
              autoComplete="off"
              value={draft.eabKey}
              placeholder={view.acme.has_eab_key ? 'Stored' : ''}
              onChange={(event) => set({ eabKey: event.target.value })}
            />
          </PreferencesGroup>
        )}
      </fieldset>

      {editable && <FormActions dirty={dirty} busy={busy} disabled={fileIncomplete || acmeIncomplete} onDiscard={() => setDraft(draftFrom(view))} />}

      <PreferencesGroup title="Available Certificates">
        {view.certificates.map((status) => <CertificateRow key={status.name} status={status} acme={view.mode === 'acme'} />)}
      </PreferencesGroup>
    </form>
  )
}
