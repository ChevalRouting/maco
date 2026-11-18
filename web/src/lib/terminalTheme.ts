export function terminalTheme() {
  const style = getComputedStyle(document.documentElement)
  return {
    background: style.getPropertyValue('--terminal-background').trim(),
    foreground: style.getPropertyValue('--terminal-foreground').trim(),
  }
}
