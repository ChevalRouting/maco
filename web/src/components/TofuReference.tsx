import { ChevronRight } from 'lucide-react'
import { CopyButton, PreferencesGroup, cn } from 'maco-ui'
import generated from '../api/tofu-schema.json'

type SchemaType = string | [string, SchemaType] | ['object', Record<string, SchemaType>]

interface SchemaAttribute {
  type?: SchemaType
  nested_type?: NestedType
  description?: string
  required?: boolean
  optional?: boolean
  computed?: boolean
  sensitive?: boolean
}

interface NestedType {
  attributes: Record<string, SchemaAttribute>
  nesting_mode: string
}

interface SchemaBlock {
  attributes?: Record<string, SchemaAttribute>
  description?: string
}

interface Schema {
  block: SchemaBlock
}

interface ProviderSchema {
  provider: Schema
  resource_schemas?: Record<string, Schema>
  data_source_schemas?: Record<string, Schema>
}

interface SchemaDocument {
  provider_schemas: Record<string, ProviderSchema>
}

type EntryKind = 'provider' | 'resource' | 'data'

interface Entry {
  kind: EntryKind
  name: string
  block: SchemaBlock
}

interface Field {
  path: string
  attribute: SchemaAttribute
  depth: number
}

const document = generated as unknown as SchemaDocument
const [address, provider] = Object.entries(document.provider_schemas)[0]
const source = address.split('/').slice(-2).join('/')
const local = address.split('/').pop() ?? 'maco'

const kindStyles: Record<EntryKind, string> = {
  provider: 'bg-warning/15 text-warning',
  resource: 'bg-success/15 text-success',
  data: 'bg-info/15 text-info',
}

const kindLabels: Record<EntryKind, string> = {
  provider: 'Provider',
  resource: 'Resource',
  data: 'Data',
}

function typeName(type?: SchemaType): string {
  if (!type) return 'unknown'
  if (typeof type === 'string') return type
  const [kind, inner] = type
  if (kind === 'object') {
    return `object({ ${Object.entries(inner as Record<string, SchemaType>).map(([name, value]) => `${name} = ${typeName(value)}`).join(', ')} })`
  }
  return `${kind}(${typeName(inner as SchemaType)})`
}

function attributeType(attribute: SchemaAttribute): string {
  if (!attribute.nested_type) return typeName(attribute.type)
  const fields = Object.entries(attribute.nested_type.attributes).map(([name, value]) => `${name} = ${attributeType(value)}`)
  const object = `object({ ${fields.join(', ')} })`
  return attribute.nested_type.nesting_mode === 'single' ? object : `${attribute.nested_type.nesting_mode}(${object})`
}

function readOnly(attribute: SchemaAttribute) {
  return !!attribute.computed && !attribute.optional && !attribute.required
}

function ordered(attributes: Record<string, SchemaAttribute> = {}) {
  const rank = (attribute: SchemaAttribute) => (attribute.required ? 0 : readOnly(attribute) ? 2 : 1)
  return Object.entries(attributes).sort(([a, left], [b, right]) => rank(left) - rank(right) || a.localeCompare(b))
}

function fields(attributes: Record<string, SchemaAttribute> = {}, prefix = '', depth = 0): Field[] {
  return ordered(attributes).flatMap(([name, attribute]) => {
    const path = prefix + name
    const nested = attribute.nested_type
    const suffix = nested && nested.nesting_mode !== 'single' ? '[].' : '.'
    return [{ path, attribute, depth }, ...(nested ? fields(nested.attributes, path + suffix, depth + 1) : [])]
  })
}

function placeholder(attribute: SchemaAttribute): string {
  const nested = attribute.nested_type
  if (nested) {
    const inner = ordered(nested.attributes)
      .filter(([, value]) => !readOnly(value))
      .map(([name, value]) => `${name} = ${placeholder(value)}`)
      .join(', ')
    return nested.nesting_mode === 'single' ? `{ ${inner} }` : `[{ ${inner} }]`
  }
  const type = attribute.type
  if (type === 'string') return '""'
  if (type === 'number') return '0'
  if (type === 'bool') return 'false'
  if (Array.isArray(type) && type[0] === 'map') return '{}'
  if (Array.isArray(type) && type[0] === 'object') return '{}'
  return '[]'
}

function usage(entry: Entry): string {
  const args = ordered(entry.block.attributes).filter(([, attribute]) => !readOnly(attribute))
  const width = Math.max(0, ...args.map(([name]) => name.length))
  const body = args.map(([name, attribute]) => `  ${name.padEnd(width)} = ${placeholder(attribute)}`).join('\n')
  const head = entry.kind === 'provider' ? `provider "${entry.name}"` : `${entry.kind === 'data' ? 'data' : 'resource'} "${entry.name}" "example"`
  return `${head} {\n${body}\n}`
}

interface FlagProps {
  className: string
  children: string
}

function Flag({ className, children }: FlagProps) {
  return <span className={cn('rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase leading-none tracking-wide', className)}>{children}</span>
}

interface FieldListProps {
  title: string
  items: Field[]
}

function FieldList({ title, items }: FieldListProps) {
  if (items.length === 0) return null

  return (
    <div className="space-y-2">
      <h4 className="text-xs font-semibold text-muted-foreground">{title}</h4>
      <div className="divide-y divide-border overflow-hidden rounded-lg bg-muted/40">
        {items.map(({ path, attribute, depth }) => (
          <div key={path} className="space-y-1 px-3 py-2.5" style={{ paddingLeft: `${0.75 + depth * 1.25}rem` }}>
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span className="font-mono text-sm font-semibold text-foreground">{path}</span>
              <span className="break-all font-mono text-xs text-muted-foreground">{attributeType(attribute)}</span>
              {attribute.required && <Flag className="bg-danger/15 text-danger">required</Flag>}
              {attribute.optional && <Flag className="bg-muted text-muted-foreground">optional</Flag>}
              {readOnly(attribute) && <Flag className="bg-info/15 text-info">read-only</Flag>}
              {attribute.sensitive && <Flag className="bg-warning/15 text-warning">sensitive</Flag>}
            </div>
            {attribute.description && <p className="text-sm text-muted-foreground">{attribute.description}</p>}
          </div>
        ))}
      </div>
    </div>
  )
}

interface EntryBlockProps {
  entry: Entry
}

function EntryBlock({ entry }: EntryBlockProps) {
  const all = fields(entry.block.attributes)
  const example = usage(entry)

  return (
    <details className="group">
      <summary className="flex min-h-[2.75rem] cursor-pointer list-none items-center gap-3 px-4 py-2 active:bg-foreground/[0.05] [&::-webkit-details-marker]:hidden">
        <span className={cn('w-20 shrink-0 rounded-md px-2 py-1 text-center text-xs font-bold uppercase', kindStyles[entry.kind])}>{kindLabels[entry.kind]}</span>
        <span className="min-w-0 shrink-0 font-mono text-sm font-semibold text-foreground">{entry.name}</span>
        <span className="hidden min-w-0 flex-1 truncate text-sm text-muted-foreground md:block">{entry.block.description}</span>
        <ChevronRight aria-hidden="true" strokeWidth={2.5} className="ml-auto h-3.5 w-3.5 shrink-0 text-muted-foreground/60 transition-transform group-open:rotate-90" />
      </summary>
      <div className="space-y-5 px-4 pb-5 pt-1">
        {entry.block.description && <p className="text-sm text-foreground">{entry.block.description}</p>}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-semibold text-muted-foreground">Example Value</h4>
            <CopyButton text={example} label={`Copy ${entry.name} Example`} />
          </div>
          <pre className="m-0 overflow-x-auto rounded-lg bg-muted/40 p-3 font-mono text-xs leading-relaxed text-foreground">{example}</pre>
        </div>
        <FieldList title="Arguments" items={all.filter((field) => !readOnly(field.attribute))} />
        <FieldList title="Attributes" items={all.filter((field) => readOnly(field.attribute))} />
      </div>
    </details>
  )
}

function entries(kind: EntryKind, schemas: Record<string, Schema> = {}): Entry[] {
  return Object.entries(schemas)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, schema]) => ({ kind, name, block: schema.block }))
}

const groups = [
  { title: 'Provider', items: [{ kind: 'provider' as const, name: local, block: provider.provider.block }] },
  { title: 'Resources', items: entries('resource', provider.resource_schemas) },
  { title: 'Data Sources', items: entries('data', provider.data_source_schemas) },
].filter((group) => group.items.length > 0)

export default function TofuReference() {
  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="space-y-1 px-1">
        <p className="font-mono text-sm font-semibold text-foreground">{source}</p>
        <p className="text-sm text-muted-foreground">Generated from the provider schema with tofu providers schema -json</p>
      </div>
      {groups.map((group) => (
        <PreferencesGroup key={group.title} title={group.title}>
          {group.items.map((entry) => (
            <EntryBlock key={`${entry.kind}.${entry.name}`} entry={entry} />
          ))}
        </PreferencesGroup>
      ))}
    </div>
  )
}
