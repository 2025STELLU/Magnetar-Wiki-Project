import { wFetch } from '../util/fetch-util'

export interface NameStyle {
  font_family: string
  color: string
  gradient_from: string
  gradient_to: string
  gradient_direction: string
}

export interface UserData {
  type: 'system' | 'anonymous' | 'bot' | 'normal' | 'wikidot'
  id?: number
  avatar?: string
  name: string
  username: string
  urlName?: string
  showAvatar: boolean
  editor?: boolean
  staff?: boolean
  admin?: boolean
  roles?: string
  name_style?: NameStyle | null
}

export function fetchAllUsers(): Promise<UserData[]> {
  return wFetch<UserData[]>('/pw-api/users')
}

export function lookupUser(username: string): Promise<UserData> {
  return wFetch<UserData>(`/pw-api/users/lookup?username=${encodeURIComponent(username)}`)
}

export interface AdminSusUser {
  user: {
    id: number
    name: string
  }
  ip: string
}

export function fetchAdminSusUsers(): Promise<AdminSusUser[]> {
  return wFetch<AdminSusUser[]>('/pw-api/admin/sus')
}
