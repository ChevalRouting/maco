import { Toaster as Sonner } from 'sonner'

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ theme, ...props }: ToasterProps) => {
  return (
    <Sonner
      theme={theme}
      position="bottom-center"
      mobileOffset={{ bottom: 'calc(4.25rem + env(safe-area-inset-bottom))' }}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            'group toast group-[.toaster]:rounded-2xl group-[.toaster]:border-0 group-[.toaster]:bg-card/80 group-[.toaster]:text-foreground group-[.toaster]:backdrop-blur-xl group-[.toaster]:backdrop-saturate-150 group-[.toaster]:[box-shadow:var(--overlay-shadow)]',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
          cancelButton: 'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground',
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
