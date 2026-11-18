export function fmtBytes(b: number): string {
  if (b >= 1e9) {
    return (b / 1e9).toFixed(1) + ' GB'
  }
  if (b >= 1e6) {
    return (b / 1e6).toFixed(1) + ' MB'
  }
  if (b >= 1e3) {
    return (b / 1e3).toFixed(1) + ' KB'
  }
  return b + ' B'
}
