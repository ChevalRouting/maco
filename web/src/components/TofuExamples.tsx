import { useEffect, useState } from 'react'
import { Button, CopyButton, PreferencesGroup, Row, Segmented } from 'maco-ui'
import { Download } from 'lucide-react'

interface CodeProps {
  title: string
  description?: string
  code: string
}

function Code({ title, description, code }: CodeProps) {
  return (
    <PreferencesGroup title={title} description={description} header={<CopyButton text={code} label={`Copy ${title}`} />}>
      <pre className="m-0 overflow-x-auto p-4 font-mono text-xs leading-relaxed text-foreground">{code}</pre>
    </PreferencesGroup>
  )
}

interface ProviderFile {
  os: string
  arch: string
  file: string
  sha256: string
}

interface ProviderRelease {
  version: string
  checksums: string
  files: ProviderFile[]
}

const platformNames: Record<string, string> = {
  linux_amd64: 'Linux x86-64',
  linux_arm64: 'Linux ARM64',
  darwin_arm64: 'macOS Apple Silicon',
}

function platformKey(file: ProviderFile) {
  return `${file.os}_${file.arch}`
}

function platformName(file: ProviderFile) {
  return platformNames[platformKey(file)] ?? `${file.os} ${file.arch}`
}

function visitorPlatform(files: ProviderFile[]) {
  const agent = navigator.userAgent
  const preferred = /Mac/.test(agent) ? 'darwin_arm64' : /aarch64|arm64/i.test(agent) ? 'linux_arm64' : 'linux_amd64'
  return files.find((file) => platformKey(file) === preferred) ?? files[0]
}

async function loadRelease(): Promise<ProviderRelease | null> {
  try {
    const response = await fetch('/downloads/terraform-provider-maco.json')
    const release = (await response.json()) as ProviderRelease
    return response.ok && release.files?.length ? release : null
  } catch {
    return null
  }
}

function installScript(endpoint: string, file: ProviderFile) {
  return `for registry in registry.opentofu.org registry.terraform.io; do
  mkdir -p ~/.terraform.d/plugins/$registry/chevalrouting/maco
  curl -fsSL -o ~/.terraform.d/plugins/$registry/chevalrouting/maco/${file.file} \\
    ${endpoint}/downloads/${file.file}
done
tofu init`
}

interface ProviderDownloadsProps {
  endpoint: string
}

function ProviderDownloads({ endpoint }: ProviderDownloadsProps) {
  const [release, setRelease] = useState<ProviderRelease | null>(null)
  const [platform, setPlatform] = useState('')

  useEffect(() => {
    void loadRelease().then((loaded) => {
      setRelease(loaded)
      if (loaded) setPlatform(platformKey(visitorPlatform(loaded.files)))
    })
  }, [])

  if (!release) return null
  const selected = release.files.find((file) => platformKey(file) === platform) ?? release.files[0]

  return (
    <>
      <PreferencesGroup title="Download" description={`Provider ${release.version} for OpenTofu and Terraform, served by this maco server`}>
        {release.files.map((file) => (
          <Row key={file.file} title={platformName(file)} subtitle={<span className="break-all font-mono">sha256 {file.sha256}</span>}>
            <Button asChild className="gap-2">
              <a href={`/downloads/${file.file}`} download>
                <Download aria-hidden="true" className="h-4 w-4" />
                Download
              </a>
            </Button>
          </Row>
        ))}
        <Row title="Checksums">
          <Button asChild variant="ghost">
            <a href={`/downloads/${release.checksums}`} download>
              SHA256SUMS
            </a>
          </Button>
        </Row>
      </PreferencesGroup>
      <PreferencesGroup
        title="Install"
        description="Places the archive in the local plugin mirror of both tools; add --cacert with the server certificate if it is self-signed"
        header={
          <div className="flex flex-wrap items-center gap-2">
            <Segmented
              value={platformKey(selected)}
              onChange={setPlatform}
              options={release.files.map((file) => ({ value: platformKey(file), label: platformName(file) }))}
            />
            <CopyButton text={installScript(endpoint, selected)} label="Copy Install" />
          </div>
        }
      >
        <pre className="m-0 overflow-x-auto p-4 font-mono text-xs leading-relaxed text-foreground">{installScript(endpoint, selected)}</pre>
      </PreferencesGroup>
    </>
  )
}

function providerBlock(endpoint: string, version?: string) {
  return `terraform {
  required_providers {
    maco = {
      source  = "chevalrouting/maco"${version ? `\n      version = "${version}"` : ''}
    }
  }
}

provider "maco" {
  endpoint = "${endpoint}"
}`
}

function credentials(endpoint: string) {
  return `export MACO_ENDPOINT=${endpoint}
export MACO_API_KEY=your-api-key
tofu plan
tofu apply`
}

const networkExample = `resource "maco_network" "lab" {
  name    = "lab"
  mode    = "bridge"
  address = "10.20.0.1/24"
  vlans   = [{ parent = "en0", tag = 100 }]
}

resource "maco_network" "isolated" {
  name = "isolated"
  mode = "switch"
}

data "maco_network" "lan" {
  name = "lan"
}`

const vmExample = `resource "maco_vm" "app" {
  name          = "app"
  image         = "ubuntu-24.04-arm64"
  cpus          = 4
  memory_mib    = 4096
  disk_size_gib = 40
  network       = "user"
  running       = true
  tags          = ["web", "prod"]

  disks = [
    { name = "data", size_gib = 50 },
  ]

  lifecycle {
    prevent_destroy = true
  }
}`

const templateExample = `resource "maco_template" "web" {
  name              = "web"
  description       = "Ubuntu web node with cloud-init"
  image             = "ubuntu-24.04-arm64"
  cpus              = 2
  memory_mib        = 2048
  disk_size_gib     = 20
  tags              = ["web"]
  guest_provisioner = "cloud-init"
  guest_data        = file("\${path.module}/web.yaml")
}`

const quickstartExample = `provider "maco" {}

resource "maco_vm" "hello" {
  name    = "hello"
  image   = "ubuntu-24.04-arm64"
  running = true

  guest_provisioner = "cloud-init"
  guest_data        = <<-YAML
    #cloud-config
    users:
      - default
    packages:
      - nginx
  YAML
}

output "hello_id" {
  value = maco_vm.hello.id
}`

const staticExample = `resource "maco_network" "private" {
  name    = "private"
  mode    = "bridge"
  address = "10.42.0.1/24"
}

resource "maco_vm" "db" {
  name        = "db"
  image       = "debian-12-arm64"
  network     = maco_network.private.id
  addresses   = ["10.42.0.10/24"]
  gateway     = "10.42.0.1"
  nameservers = ["1.1.1.1", "9.9.9.9"]
  running     = true

  guest_provisioner = "cloud-init"
  guest_data        = file("\${path.module}/network.yaml")
}`

const staticCloudInit = `#cloud-config
hostname: [[ .Name ]]
write_files:
  - path: /etc/netplan/60-static.yaml
    permissions: "0600"
    content: |
      network:
        version: 2
        ethernets:
          primary:
            match:
              macaddress: "[[ (index .Interfaces 0).MAC ]]"
            dhcp4: false
            addresses:
            [[- range .Addresses ]]
              - [[ . ]]
            [[- end ]]
runcmd:
  - netplan apply`

const fleetExample = `locals {
  workers = {
    "worker-1" = "10.42.0.21/24"
    "worker-2" = "10.42.0.22/24"
    "worker-3" = "10.42.0.23/24"
  }
}

data "maco_network" "lan" {
  name = "lan"
}

resource "maco_vm" "worker" {
  for_each = local.workers

  name          = each.key
  image         = "ubuntu-24.04-arm64"
  cpus          = 4
  memory_mib    = 8192
  network       = data.maco_network.lan.id
  addresses     = [each.value]
  gateway       = "10.42.0.1"
  nameservers   = ["10.42.0.1"]
  tags          = ["k8s", "worker"]
  autostart     = true
  running       = true

  disks = [
    { name = "containers", size_gib = 100 },
  ]
}`

const flatcarExample = `resource "maco_vm" "edge" {
  name    = "edge"
  image   = "flatcar-stable-arm64"
  running = true

  guest_provisioner = "ignition"
  guest_data        = <<-YAML
    variant: flatcar
    version: 1.0.0
    passwd:
      users:
        - name: core
          ssh_authorized_keys:
            - ssh-ed25519 AAAA... you@example
  YAML
}`

const importExample = `tofu import maco_network.private NETWORK_UUID
tofu import maco_vm.app VM_UUID
tofu import maco_template.web TEMPLATE_UUID`

function useProviderVersion() {
  const [version, setVersion] = useState<string>()
  useEffect(() => {
    void loadRelease().then((release) => setVersion(release?.version))
  }, [])
  return version
}

export default function TofuExamples() {
  const endpoint = window.location.origin
  const providerVersion = useProviderVersion()

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <ProviderDownloads endpoint={endpoint} />
      <Code title="Provider" description="Point the provider at this maco server" code={providerBlock(endpoint, providerVersion)} />
      <Code title="Credentials" description="Prefer environment variables, with an administrator key from API Keys" code={credentials(endpoint)} />
      <Code title="Quickstart" description="A NAT guest with nginx, using MACO_ENDPOINT and MACO_API_KEY" code={quickstartExample} />
      <Code title="Networks" description="A bridge with a VLAN, an isolated switch, and a lookup of a host network" code={networkExample} />
      <Code title="Static Addressing" description="A bridge with a host address and a guest with a fixed IP" code={staticExample} />
      <Code title="network.yaml" description="Cloud-init loaded with file(); maco expands the [[ ]] placeholders" code={staticCloudInit} />
      <Code title="Data Disks" description="CPU and memory changes need running = false in the same apply" code={vmExample} />
      <Code title="Fleet" description="Several workers on an existing host network with for_each" code={fleetExample} />
      <Code title="Flatcar" description="Butane config provisioned through Ignition" code={flatcarExample} />
      <Code title="Template" description="A saved profile that pre-fills VM creation" code={templateExample} />
      <Code title="Import" description="Adopt existing resources by UUID" code={importExample} />
    </div>
  )
}
