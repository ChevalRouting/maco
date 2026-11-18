import { forwardRef, useImperativeHandle, useRef, type FormEvent, type KeyboardEvent } from 'react'
import { Button } from 'maco-ui'
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp } from 'lucide-react'

export interface KeySink {
  sendKey: (keysym: number, down?: boolean) => void
  sendCtrlAltDel: () => void
}

export interface ConsoleKeyboardHandle {
  open: () => void
}

interface ConsoleKeyboardProps {
  sink: KeySink | null
  visible: boolean
}

const buffer = '__'

function reset(field: HTMLTextAreaElement) {
  field.value = buffer
  field.setSelectionRange(buffer.length, buffer.length)
}

const keysyms: Record<string, number> = {
  Backspace: 0xff08,
  Tab: 0xff09,
  Enter: 0xff0d,
  Escape: 0xff1b,
  ArrowLeft: 0xff51,
  ArrowUp: 0xff52,
  ArrowRight: 0xff53,
  ArrowDown: 0xff54,
  Delete: 0xffff,
}

const extraKeys = [
  { key: 'Escape', label: 'Esc' },
  { key: 'Tab', label: 'Tab' },
  { key: 'ArrowLeft', label: 'Left', icon: ArrowLeft },
  { key: 'ArrowUp', label: 'Up', icon: ArrowUp },
  { key: 'ArrowDown', label: 'Down', icon: ArrowDown },
  { key: 'ArrowRight', label: 'Right', icon: ArrowRight },
]

const shift = 0xffe1
const shifted = /^[A-Z~!@#$%^&*()_+{}|:"<>?]$/

function typeChar(sink: KeySink, char: string) {
  const keysym = charKeysym(char)
  if (!shifted.test(char)) {
    sink.sendKey(keysym)
    return
  }
  sink.sendKey(shift, true)
  sink.sendKey(keysym)
  sink.sendKey(shift, false)
}

function charKeysym(char: string): number {
  if (char === '\n') return keysyms.Enter
  const code = char.codePointAt(0) ?? 0
  return code < 0x100 ? code : 0x01000000 + code
}

export const ConsoleKeyboard = forwardRef<ConsoleKeyboardHandle, ConsoleKeyboardProps>(function ConsoleKeyboard({ sink, visible }, ref) {
  const input = useRef<HTMLTextAreaElement>(null)

  useImperativeHandle(ref, () => ({
    open: () => {
      if (!input.current) return
      input.current.focus()
      reset(input.current)
    },
  }))

  function keyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    const keysym = keysyms[event.key]
    if (!keysym || !sink) return
    event.preventDefault()
    event.stopPropagation()
    sink.sendKey(keysym)
  }

  function changed(event: FormEvent<HTMLTextAreaElement>) {
    const field = event.currentTarget
    const value = field.value
    if (sink) {
      if (value.length < buffer.length) {
        for (let i = value.length; i < buffer.length; i++) sink.sendKey(keysyms.Backspace)
      } else {
        for (const char of value.slice(buffer.length)) typeChar(sink, char)
      }
    }
    reset(field)
  }

  function press(key: string) {
    sink?.sendKey(keysyms[key])
    input.current?.focus()
  }

  return (
    <div data-console-input className={visible ? 'space-y-2' : undefined}>
      <textarea
        ref={input}
        aria-label="Console Keyboard Input"
        autoCapitalize="off"
        autoComplete="off"
        autoCorrect="off"
        spellCheck={false}
        defaultValue={buffer}
        onKeyDown={keyDown}
        onInput={changed}
        onFocus={(event) => reset(event.currentTarget)}
        className="pointer-events-none fixed left-0 top-0 h-px w-px resize-none opacity-0"
      />
      {visible && (
        <div className="flex flex-wrap gap-1.5">
          {extraKeys.map(({ key, label, icon: Icon }) => (
            <Button
              key={key}
              type="button"
              variant="outline"
              size="sm"
              aria-label={label}
              title={label}
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => press(key)}
            >
              {Icon ? <Icon aria-hidden="true" className="h-4 w-4" /> : label}
            </Button>
          ))}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => {
              sink?.sendCtrlAltDel()
              input.current?.focus()
            }}
          >
            Ctrl+Alt+Del
          </Button>
        </div>
      )}
    </div>
  )
})
