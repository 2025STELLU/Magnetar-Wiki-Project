import * as React from 'react'
import { NameStyle, UserData } from '../api/user'

export const ANON_AVATAR = '/-/static/images/anon_avatar.png'
export const DEFAULT_AVATAR = '/-/static/images/default_avatar.png'
export const WIKIDOT_AVATAR = '/-/static/images/wikidot_avatar.png'

// 管理员给用户名配的字体/颜色/渐变，转成行内样式。渐变优先于纯色。
export function nameStyleProps(ns?: NameStyle | null): React.CSSProperties {
  const style: React.CSSProperties = {}
  if (!ns) return style
  if (ns.font_family) style.fontFamily = ns.font_family
  if (ns.gradient_from && ns.gradient_to) {
    style.background = `linear-gradient(${ns.gradient_direction || 'to right'}, ${ns.gradient_from}, ${ns.gradient_to})`
    style.WebkitBackgroundClip = 'text'
    style.backgroundClip = 'text'
    style.color = 'transparent'
  } else if (ns.color) {
    style.color = ns.color
  }
  return style
}

interface Props {
  data: UserData
  avatarHover?: boolean
  hideAvatar?: boolean
}

const UserView: React.FC<Props> = ({ data, avatarHover, hideAvatar }) => {
  if (data.type === 'system') {
    return (
      <span className={`printuser ${avatarHover !== false ? 'avatarhover' : ''}`}>
        <strong>system</strong>
      </span>
    )
  }
  if (data.type === 'anonymous') {
    return (
      <span className={`printuser ${avatarHover !== false ? 'avatarhover' : ''}`}>
        {data.showAvatar && !hideAvatar && (
          <span>
            <img className="small" src={ANON_AVATAR} alt="Anonymous User" />
          </span>
        )}
        <span>Anonymous User</span>
      </span>
    )
  }

  let avatar = data.type === 'wikidot' ? WIKIDOT_AVATAR : data.avatar || DEFAULT_AVATAR

  return (
    <span className={`printuser w-user ${avatarHover !== false ? 'avatarhover' : ''}`} data-user-name={data.username}>
      {data.showAvatar && !hideAvatar && (
        <a href={`/-/users/${data.id}-${encodeURIComponent(data.username)}`}>
          <img className="small" src={avatar} alt={data.name} />
        </a>
      )}
      &nbsp;
      <a href={`/-/users/${data.id}-${encodeURIComponent(data.username)}`} style={nameStyleProps(data.name_style)}>
        {data.name}
      </a>
    </span>
  )
}

export default UserView
