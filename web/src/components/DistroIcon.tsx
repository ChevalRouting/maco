const known = new Set(['ubuntu', 'debian', 'fedora', 'rocky-linux', 'almalinux', 'opensuse', 'alpine'])

interface DistroIconProps {
  distro: string
  className?: string
}

export function DistroIcon({ distro, className = '' }: DistroIconProps) {
  const slug = known.has(distro) ? distro : 'tux'
  return <i aria-hidden="true" className={`fl-${slug} ${className}`} />
}
