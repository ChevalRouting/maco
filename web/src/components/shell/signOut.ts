import { clearToken } from '../../api'

export function signOut() {
  clearToken()
  Object.keys(sessionStorage)
    .filter((key) => key.startsWith('maco:draft:'))
    .forEach((key) => sessionStorage.removeItem(key))
  window.location.assign('/login')
}
