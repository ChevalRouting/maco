import { client } from './api/generated/client.gen'
import { DefaultApi } from './api/generated'
import type {
  ApiCurrentUser,
  ApiGuestAgentResponse,
  ApiPageEngineDiskView,
  EngineBackupInfo,
  EngineCatalogImage,
  EngineCreateNetworkParams,
  EngineCreateVmParams,
  EngineCreateDiskParams,
  EngineGuestSetupParams,
  EngineGuestSetupView,
  TypesGuestCapability,
  EngineDiskView,
  EngineHostInfo,
  EngineInterfaceParams,
  EngineMedia,
  EngineStorageStats,
  EngineTemplate,
  ApiTemplateRequest,
  EngineUpdateHardwareParams,
  EngineUsbAttachment,
  EngineUsbInventory,
  EngineVmView,
  HostPort,
  JobsJob,
  JobsPage,
  TypesApiKey,
  TypesSshKey,
  TypesBackupSchedule,
  TypesNetworkManifest,
  TypesVlan,
  TypesVmDisk,
  TypesVmInterface,
  TypesVmManifest,
  UsbDevice,
  VmSnapshot,
  VmGuestFilesystem,
  VmGuestInterface,
  VmGuestIpAddress,
  VmGuestOsInfo,
} from './api/generated/types.gen'

const TOKEN_KEY = 'maco-token'

export function getToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function clearToken() {
  sessionStorage.removeItem(TOKEN_KEY)
}

client.setConfig({ baseUrl: window.location.origin })

client.interceptors.request.use((request) => {
  const token = getToken()
  if (token) request.headers.set('Authorization', `Bearer ${token}`)
  return request
})

client.interceptors.response.use((response, request) => {
  if (response.status === 401 && new URL(request.url).pathname !== '/api/login') {
    clearToken()
    window.location.assign(
      `/login?expired=1&returnTo=${encodeURIComponent(window.location.pathname + window.location.search)}`,
    )
    throw new Error('Your session has expired')
  }
  return response
})

const api = new DefaultApi()

export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (error && typeof error === 'object' && 'error' in error) {
    const detail = (error as { error?: unknown }).error
    if (typeof detail === 'string') return detail
  }
  return String(error)
}

async function call<T>(result: Promise<{ data: T }>): Promise<T> {
  try {
    return (await result).data
  } catch (error) {
    throw new Error(errorMessage(error))
  }
}

export type CurrentUser = ApiCurrentUser
export type VMManifest = TypesVmManifest
export type CreateVMParams = EngineCreateVmParams
export type CreateDiskParams = EngineCreateDiskParams
export type GuestSetupParams = EngineGuestSetupParams
export type GuestSetupView = EngineGuestSetupView
export type GuestCapability = TypesGuestCapability
export type VMView = EngineVmView
export type Network = TypesNetworkManifest
export type VLAN = TypesVlan
export type CreateNetworkParams = EngineCreateNetworkParams
export type Job = JobsJob
export type APIKey = TypesApiKey
export type SSHKey = TypesSshKey
export type GuestOSInfo = VmGuestOsInfo
export type GuestIPAddress = VmGuestIpAddress
export type GuestInterface = VmGuestInterface
export type GuestFilesystem = VmGuestFilesystem
export type GuestAgent = ApiGuestAgentResponse
export type CatalogImage = EngineCatalogImage
export type { JobsPage }
export type UpdateHardwareParams = EngineUpdateHardwareParams
export type VMDisk = TypesVmDisk
export type DiskView = EngineDiskView
export type StorageStats = EngineStorageStats
export type HostInfo = EngineHostInfo
export type NetworkInterface = HostPort
export type USBDevice = UsbDevice
export type USBInventory = EngineUsbInventory
export type USBAttachment = EngineUsbAttachment
export type Media = EngineMedia
export type Template = EngineTemplate
export type TemplateRequest = ApiTemplateRequest
export type VMInterface = TypesVmInterface
export type InterfaceParams = EngineInterfaceParams
export type BackupInfo = EngineBackupInfo
export type Snapshot = VmSnapshot
export type BackupSchedule = TypesBackupSchedule
export interface ScheduleParams {
  enabled: boolean
  interval_hours: number
  keep_last: number
  max_age_days: number
}

export function isAdmin(user: CurrentUser | null): boolean {
  return user?.role === 'admin'
}

export function jobStateLabel(state: Job['state']) {
  const labels = {
    pending: 'Pending',
    running: 'In progress',
    succeeded: 'Succeeded',
    failed: 'Failed',
  }
  return labels[state]
}

export function getMe() {
  return call(api.currentUser())
}

export async function login(username: string, password: string): Promise<void> {
  const result = await call(api.handleLogin({ body: { username, password } }))
  sessionStorage.setItem(TOKEN_KEY, result.token)
}

export async function changePassword(current_password: string, new_password: string): Promise<void> {
  const result = await call(api.changePassword({ body: { current_password, new_password } }))
  if (result.token) sessionStorage.setItem(TOKEN_KEY, result.token)
}

export function listAPIKeys() {
  return call(api.listApiKeys())
}

export function createAPIKey(name: string) {
  return call(api.createApiKey({ body: { name } }))
}

export function revokeAPIKey(id: string) {
  return call(api.revokeApiKey({ path: { id } }))
}

export function listVMs() {
  return call(api.listVms())
}

export async function getVM(id: string): Promise<VMView | null> {
  const { data, error, response } = await api.getVm({ path: { id }, throwOnError: false })
  if (response?.status === 404) return null
  if (!response?.ok) throw new Error(errorMessage(error))

  const vm = data as VMView
  if (vm.phase !== 'running' || vm.boot_time > 0) return vm

  const list = await listVMs()
  return list.find((entry) => entry.manifest.id === vm.manifest.id) || vm
}

export function createVM(body: CreateVMParams) {
  return call(api.createVm({ body }))
}

export function listSSHKeys() {
  return call(api.listSshKeys())
}

export function addSSHKey(name: string, public_key: string) {
  return call(api.addSshKey({ body: { name, public_key } }))
}

export function deleteSSHKey(id: string) {
  return call(api.deleteSshKey({ path: { id } }))
}

export function suggestMAC() {
  return call(api.suggestMac())
}

export function getGuestAgent(id: string) {
  return call(api.vmGuestAgent({ path: { id } }))
}

export function startVM(id: string) {
  return call(api.startVm({ path: { id } }))
}

export function stopVM(id: string) {
  return call(api.stopVm({ path: { id } }))
}

export function shutdownVM(id: string) {
  return call(api.shutdownVm({ path: { id } }))
}

export function rebootVM(id: string) {
  return call(api.rebootVm({ path: { id } }))
}

export function deleteVM(id: string) {
  return call(api.deleteVm({ path: { id } }))
}

export function listBackups(id: string) {
  return call(api.listBackups({ path: { id } }))
}

export function createBackup(id: string): Promise<Job> {
  return call(api.createBackup({ path: { id } }))
}

export function restoreBackup(id: string, timestamp: string, asNew: boolean): Promise<Job> {
  return call(api.restoreBackup({ path: { id, timestamp }, body: { as_new: asNew } }))
}

export function deleteBackup(id: string, timestamp: string): Promise<Job> {
  return call(api.deleteBackup({ path: { id, timestamp } }))
}

export function listSnapshots(id: string) {
  return call(api.listSnapshots({ path: { id } }))
}

export function createSnapshot(id: string, tag: string, includeRAM: boolean): Promise<Job> {
  return call(api.createSnapshot({ path: { id }, body: { tag, include_ram: includeRAM } }))
}

export function restoreSnapshot(id: string, tag: string): Promise<Job> {
  return call(api.restoreSnapshot({ path: { id, tag } }))
}

export function deleteSnapshot(id: string, tag: string): Promise<Job> {
  return call(api.deleteSnapshot({ path: { id, tag } }))
}

export function getBackupSchedule(id: string) {
  return call(api.getBackupSchedule({ path: { id } }))
}

export function setBackupSchedule(id: string, params: ScheduleParams) {
  return call(api.setBackupSchedule({ path: { id }, body: params }))
}

export function listNetworks() {
  return call(api.listNetworks())
}

export function createNetwork(body: CreateNetworkParams) {
  return call(api.createNetwork({ body }))
}

export function updateNetwork(id: string, body: CreateNetworkParams) {
  return call(api.updateNetwork({ path: { id }, body }))
}

export function applyNetwork(id: string) {
  return call(api.applyNetwork({ path: { id } }))
}

export function destroyNetwork(id: string) {
  return call(api.destroyNetwork({ path: { id } }))
}

export function listImages() {
  return call(api.listImages())
}

export function listCatalog() {
  return call(api.listCatalog())
}

export function downloadCatalogImage(id: string) {
  return call(api.downloadCatalogImage({ path: { id } }))
}

export function deleteCatalogImage(id: string) {
  return call(api.deleteCatalogImage({ path: { id } }))
}

export function listJobs(page: number, state: string, search: string, focus = '') {
  return call(api.listJobs({ query: { page, page_size: 25, state, search, focus } }))
}

export function getJob(id: string) {
  return call(api.getJob({ path: { id } }))
}

export function updateHardware(id: string, params: UpdateHardwareParams): Promise<Job> {
  return call(api.updateHardware({ path: { id }, body: params }))
}

export function getGuestSetup(id: string): Promise<GuestSetupView> {
  return call(api.getGuestSetup({ path: { id } }))
}

export function updateGuestSetup(id: string, params: GuestSetupParams): Promise<Job> {
  return call(api.updateGuestSetup({ path: { id }, body: params }))
}

export function addDisk(id: string, name: string, size: number): Promise<Job> {
  return call(api.addDisk({ path: { id }, body: { name, size_gib: size } }))
}

export function growDisk(id: string, disk: string, size: number): Promise<Job> {
  return call(api.growDisk({ path: { id, disk }, body: { size_gib: size } }))
}

export function removeDisk(id: string, disk: string): Promise<Job> {
  return call(api.removeDisk({ path: { id, disk } }))
}

export function attachImage(id: string, media: Media) {
  return call(api.addDisk({ path: { id }, body: { name: media.name, size_gib: media.size_gib, image_id: media.id } }))
}

export type DiskPage = ApiPageEngineDiskView

export function listDisks(page = 1) {
  return call(api.listDisks({ query: { page, page_size: 25 } }))
}

export function getDiskStorage() {
  return call(api.diskStorage())
}

export function getHost() {
  return call(api.hostInfo())
}

export function powerOffHost(force: boolean) {
  return call(api.powerOffHost({ body: { force } }))
}

export function rebootHost(force: boolean) {
  return call(api.rebootHost({ body: { force } }))
}

export function listInterfaces() {
  return call(api.listInterfaces())
}

export function listUSBDevices() {
  return call(api.listUsbDevices())
}

export function listVMUSB(id: string) {
  return call(api.listVmusb({ path: { id } }))
}

export function attachUSB(id: string, device: USBDevice) {
  return call(api.attachUsb({ path: { id }, body: { device_id: device.id, fingerprint: device.fingerprint } }))
}

export function detachUSB(id: string, attachment: string) {
  return call(api.detachUsb({ path: { id, attachment } }))
}

export function assignUSB(id: string, device: USBDevice) {
  return call(api.assignUsb({ path: { id }, body: { device_id: device.id, fingerprint: device.fingerprint } }))
}

export function unassignUSB(id: string, key: string) {
  return call(api.unassignUsb({ path: { id, key } }))
}

export function addVMInterface(id: string, params: InterfaceParams) {
  return call(api.addVmInterface({ path: { id }, body: params }))
}

export function updateVMInterface(id: string, nic: string, params: InterfaceParams) {
  return call(api.updateVmInterface({ path: { id, interface: nic }, body: params }))
}

export function removeVMInterface(id: string, nic: string) {
  return call(api.removeVmInterface({ path: { id, interface: nic } }))
}

export function listMedia() {
  return call(api.listMedia())
}

export function createMedia(name: string, size_gib: number) {
  return call(api.createMedia({ body: { name, size_gib } }))
}

export function deleteMedia(id: string) {
  return call(api.deleteMedia({ path: { id } }))
}

export function updateMedia(id: string, isos: string[], boot_order: string[]) {
  return call(api.updateMedia({ path: { id }, body: { isos, boot_order } }))
}

export function listTemplates() {
  return call(api.listTemplates())
}

export function createTemplate(body: TemplateRequest) {
  return call(api.createTemplate({ body }))
}

export function updateTemplate(id: string, body: TemplateRequest) {
  return call(api.updateTemplate({ path: { id }, body }))
}

export function deleteTemplate(id: string) {
  return call(api.deleteTemplate({ path: { id } }))
}

function uploadMedia(kind: 'image' | 'iso', file: File, onProgress?: (fraction: number) => void) {
  const form = new FormData()
  form.append('file', file)

  const uploadFetch = (request: Request): Promise<Response> =>
    new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open(request.method, request.url)
      request.headers.forEach((value, name) => {
        if (name.toLowerCase() === 'content-type') return
        xhr.setRequestHeader(name, value)
      })
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable) onProgress?.(event.loaded / event.total)
      }
      xhr.onload = () =>
        resolve(
          new Response(xhr.responseText, {
            status: xhr.status,
            statusText: xhr.statusText,
            headers: { 'Content-Type': 'application/json' },
          }),
        )
      xhr.onerror = () => reject(new Error('Upload failed'))
      xhr.send(form)
    })

  const options = {
    body: { file },
    bodySerializer: () => form,
    fetch: uploadFetch as unknown as typeof globalThis.fetch,
  }
  return call(kind === 'image' ? api.uploadImage(options) : api.uploadIso(options))
}

export function uploadISO(file: File, onProgress?: (fraction: number) => void) {
  return uploadMedia('iso', file, onProgress)
}

export function uploadImage(file: File, onProgress?: (fraction: number) => void) {
  return uploadMedia('image', file, onProgress)
}

export async function getVMPreview(
  id: string,
  signal: AbortSignal,
): Promise<{ data: Blob | undefined; response: Response | undefined }> {
  const result = await api.previewVm({ path: { id }, signal, parseAs: 'blob', throwOnError: false })
  return { data: result.data as Blob | undefined, response: result.response }
}
