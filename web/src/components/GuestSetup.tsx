import { useEffect } from 'react'
import { Tabs, Textarea } from 'maco-ui'
import { ChoiceRow } from './ChoiceRow'
import { type GuestCapability } from '../api'
import { TemplateVariables } from './TemplateVariables'

export interface GuestFile {
  name: string
  content: string
}

export interface GuestSetupValue {
  provisioner: string
  files: GuestFile[]
}

const provisionerLabels: Record<string, string> = {
  'cloud-init': 'cloud-init',
  ignition: 'Ignition (Butane)',
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

interface GuestSetupProps {
  value: GuestSetupValue
  onChange: (value: GuestSetupValue) => void
  capabilities: GuestCapability[]
  disabled?: boolean
  activeFile: string
  onActiveFileChange: (name: string) => void
}

export function GuestSetup({ value, onChange, capabilities, disabled, activeFile, onActiveFileChange }: GuestSetupProps) {
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

  const fileNames = value.files.map((file) => file.name)
  const currentFile = fileNames.includes(activeFile) ? activeFile : fileNames[0] || ''
  const currentContent = value.files.find((file) => file.name === currentFile)?.content ?? ''

  function editFile(content: string) {
    onChange({
      ...value,
      files: value.files.map((file) => (file.name === currentFile ? { ...file, content } : file)),
    })
  }

  return (
    <>
      {provisioners.length > 1 && (
        <ChoiceRow
          title="Provisioner"
          value={value.provisioner}
          choices={provisioners.map((item) => ({ value: item, label: provisionerLabels[item] || item }))}
          disabled={disabled}
          onChange={(provisioner) => onChange({ ...value, provisioner, files: [starterFor(provisioner)] })}
        />
      )}
      <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
        {value.files.length > 1 ? (
          <Tabs
            id="guest-files"
            label="Configuration Files"
            tabs={value.files.map((file) => ({ key: file.name, label: file.name }))}
            active={currentFile}
            onChange={onActiveFileChange}
          />
        ) : (
          <label htmlFor="guest-file" className="font-mono text-xs font-semibold text-muted-foreground">
            {currentFile || primaryFileName(value.provisioner)}
          </label>
        )}
        <Textarea
          id="guest-file"
          aria-label={currentFile || primaryFileName(value.provisioner)}
          disabled={disabled || !currentFile}
          value={currentContent}
          spellCheck={false}
          rows={18}
          className="font-mono text-xs"
          onChange={(event) => editFile(event.target.value)}
        />
      </div>
      <TemplateVariables />
    </>
  )
}
