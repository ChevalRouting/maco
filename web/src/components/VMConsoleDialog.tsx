import { lazy, Suspense } from 'react'
import { Dialog } from 'maco-ui'

export type ConsoleMode = 'serial' | 'graphical'

interface VMConsoleDialogProps {
  id: string
  name: string
  mode: ConsoleMode
  onClose: () => void
}

const SerialConsole = lazy(loadSerial)
const GraphicalConsole = lazy(loadGraphical)

async function loadSerial() {
  const module = await import('./SerialConsole')
  return { default: module.SerialConsole }
}

async function loadGraphical() {
  const module = await import('./GraphicalConsole')
  return { default: module.GraphicalConsole }
}

export function VMConsoleDialog({
  id,
  name,
  mode,
  onClose,
}: VMConsoleDialogProps) {
  return (
    <Dialog
      open
      captureKeyboard
      onClose={onClose}
      title={`${name} ${mode === 'serial' ? 'Serial' : 'Graphical'} Console`}
      className="w-[calc(100vw-2rem)] max-w-5xl"
    >
      <Suspense fallback={<p role="status">Loading console…</p>}>
        {mode === 'serial' ? (
          <SerialConsole id={id} autoConnect />
        ) : (
          <GraphicalConsole id={id} autoConnect />
        )}
      </Suspense>
    </Dialog>
  )
}
