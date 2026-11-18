interface LoginScreenProps {
  title?: string
  subtitle?: string
  logo?: React.ComponentType<{ className?: string }>
  brand?: React.ReactNode
  children: React.ReactNode
}

export function LoginScreen({ title = 'App', subtitle, logo, brand, children }: LoginScreenProps) {
  const Logo = logo

  return (
    <main
      className="login-backdrop relative grid min-h-dvh place-items-center overflow-hidden px-4 py-10 text-foreground"
    >
      <div
        className="relative w-full max-w-[22rem]"
      >
        <header
          className="mb-7 flex flex-col items-center text-center"
        >
          {brand ?? (
            <>
              {Logo && (
                <span
                  className="mb-4 grid h-[5.25rem] w-[5.25rem] place-items-center rounded-[1.375rem] bg-card/80 backdrop-blur-xl backdrop-saturate-150 [box-shadow:var(--overlay-shadow)]"
                >
                  <Logo
                    className="h-14 w-14"
                  />
                </span>
              )}
              <h1
                className="text-[28px] font-bold leading-tight tracking-[-0.022em]"
              >
                {title}
              </h1>
              {subtitle && (
                <p
                  className="mt-1 text-sm text-muted-foreground"
                >
                  {subtitle}
                </p>
              )}
            </>
          )}
        </header>
        {children}
      </div>
    </main>
  )
}
