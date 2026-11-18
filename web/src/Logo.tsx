interface LogoProps {
  className?: string
}

export function Logo({ className }: LogoProps) {
  return (
    <svg
      className={className}
      viewBox="0 0 64 64"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="maco"
    >
      <rect
        x="5"
        y="8"
        width="54"
        height="48"
        rx="5"
        fill="#16181d"
        stroke="#2d3a4a"
        strokeWidth="1"
      />
      <g>
        <rect x="9" y="12" width="21" height="18" rx="2.5" fill="#f8fafc" />
        <rect x="9" y="17" width="21" height="0.9" fill="#cbd5e1" />
        <circle cx="12.2" cy="14.6" r="1.1" fill="#34d399" />
        <rect
          x="12"
          y="20.5"
          width="15"
          height="1.1"
          rx="0.55"
          fill="#cbd5e1"
        />
        <rect x="12" y="23.5" width="9" height="1.1" rx="0.55" fill="#e2e8f0" />
      </g>
      <g>
        <rect x="34" y="12" width="21" height="18" rx="2.5" fill="#f8fafc" />
        <rect x="34" y="17" width="21" height="0.9" fill="#cbd5e1" />
        <circle cx="37.2" cy="14.6" r="1.1" fill="#34d399" />
        <rect
          x="37"
          y="20.5"
          width="15"
          height="1.1"
          rx="0.55"
          fill="#cbd5e1"
        />
        <rect x="37" y="23.5" width="9" height="1.1" rx="0.55" fill="#e2e8f0" />
      </g>
      <g>
        <rect x="9" y="34" width="21" height="18" rx="2.5" fill="#f8fafc" />
        <rect x="9" y="39" width="21" height="0.9" fill="#cbd5e1" />
        <circle cx="12.2" cy="36.6" r="1.1" fill="#fbbf24" />
        <rect
          x="12"
          y="42.5"
          width="15"
          height="1.1"
          rx="0.55"
          fill="#cbd5e1"
        />
        <rect x="12" y="45.5" width="9" height="1.1" rx="0.55" fill="#e2e8f0" />
      </g>
      <g>
        <rect x="34" y="34" width="21" height="18" rx="2.5" fill="#f8fafc" />
        <rect x="34" y="39" width="21" height="0.9" fill="#cbd5e1" />
        <circle cx="37.2" cy="36.6" r="1.1" fill="#34d399" />
        <rect
          x="37"
          y="42.5"
          width="15"
          height="1.1"
          rx="0.55"
          fill="#cbd5e1"
        />
        <rect x="37" y="45.5" width="9" height="1.1" rx="0.55" fill="#e2e8f0" />
      </g>
    </svg>
  )
}
