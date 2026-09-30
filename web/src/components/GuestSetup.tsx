import { useEffect } from 'react'
import { PreferencesGroup, Tabs } from 'cheval-ui'
import { ChoiceRow } from './ChoiceRow'
import { type GuestCapability } from '../api'

export interface GuestFile {
  name: string
  content: string
}

export interface GuestSetupValue {
  provisioner: string
  files: GuestFile[]
}

const provisionerLabels: Record<string, string> = {
  'cloud-init': 'Cloud-init',
  ignition: 'Flatcar (Butane / Ignition)',
}

const cloudInitStarter = `#cloud-config
hostname: [[ .Name ]]
users:
  - name: maco
    groups: [sudo]
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    shell: /bin/bash
    ssh_authorized_keys:
[[- range .SSHKeys ]]
      - [[ . ]]
[[- end ]]
package_update: true
packages:
  - qemu-guest-agent
runcmd:
  - [systemctl, enable, --now, qemu-guest-agent]
`

const ignitionStarter = `variant: flatcar
version: 1.1.0
passwd:
  users:
    - name: core
      groups:
        - sudo
      ssh_authorized_keys:
[[- range .SSHKeys ]]
        - [[ . ]]
[[- end ]]
storage:
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: |
          [[ .Name ]]
    - path: /etc/flatcar/enabled-sysext.conf
      mode: 0644
      contents:
        inline: |
          podman
systemd:
  units:
    - name: podman.socket
      enabled: true
`

function primaryFileName(provisioner: string) {
  return provisioner === 'ignition' ? 'config.bu' : 'user-data'
}

function starterFor(provisioner: string): GuestFile {
  return {
    name: primaryFileName(provisioner),
    content: provisioner === 'ignition' ? ignitionStarter : cloudInitStarter,
  }
}

export function GuestSetup({
  value,
  onChange,
  capabilities,
  disabled,
  activeFile,
  onActiveFileChange,
}: {
  value: GuestSetupValue
  onChange: (value: GuestSetupValue) => void
  capabilities: GuestCapability[]
  disabled?: boolean
  activeFile: string
  onActiveFileChange: (name: string) => void
}) {
  const provisioners = capabilities.map((cap) => cap.provisioner)

  useEffect(() => {
    if (!capabilities.length) return
    let next = value
    if (!provisioners.includes(next.provisioner)) {
      next = { ...next, provisioner: provisioners[0] }
    }
    if (next.files.length === 0) {
      next = { ...next, files: [starterFor(next.provisioner)] }
    }
    if (next !== value) onChange(next)
  }, [capabilities, provisioners, value, onChange])

  const flatcar = value.provisioner === 'ignition'
  const fileNames = value.files.map((file) => file.name)
  const currentFile = fileNames.includes(activeFile) ? activeFile : fileNames[0] || ''
  const currentContent = value.files.find((file) => file.name === currentFile)?.content ?? ''

  return (
    <PreferencesGroup
      title="Guest Setup"
      description="The guest configuration delivered on first boot. Both cloud-init and Ignition only run once, on the first start of a fresh disk."
    >
      {provisioners.length > 1 && (
        <ChoiceRow
          stacked
          title="Provisioner"
          value={value.provisioner}
          choices={provisioners.map((item) => ({
            value: item,
            label: provisionerLabels[item] || item,
          }))}
          disabled={disabled}
          onChange={(provisioner) =>
            onChange({ ...value, provisioner, files: [starterFor(provisioner)] })
          }
        />
      )}

      <div className="flex min-w-0 flex-col gap-2 px-4 py-2">
        {value.files.length > 1 && (
          <Tabs
            id="guest-files"
            label="Configuration files"
            tabs={value.files.map((file) => ({ key: file.name, label: file.name }))}
            active={currentFile}
            onChange={onActiveFileChange}
          />
        )}
        <label htmlFor="guest-file" className="text-xs font-medium text-muted-foreground">
          {currentFile || (flatcar ? 'config.bu' : 'user-data')}
        </label>
        <textarea
          id="guest-file"
          disabled={disabled || !currentFile}
          value={currentContent}
          spellCheck={false}
          rows={18}
          className="w-full rounded-lg border bg-card p-3 font-mono text-xs"
          onChange={(event) =>
            onChange({
              ...value,
              files: value.files.map((file) =>
                file.name === currentFile ? { ...file, content: event.target.value } : file,
              ),
            })
          }
        />
        <p className="text-xs text-muted-foreground">
          {flatcar
            ? 'Butane is transpiled to Ignition and delivered over fw_cfg.'
            : 'Each file is written verbatim into the NoCloud seed. Everything the guest receives is editable here.'}
        </p>
      </div>
    </PreferencesGroup>
  )
}
