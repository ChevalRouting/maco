import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'

const checkedExtensions = /\.(?:cjs|css|json|md|mjs|ts|tsx|ya?ml)$/
const files = execFileSync('git', ['ls-files', '--cached', '--others', '--exclude-standard'], {
  encoding: 'utf8',
})
  .trim()
  .split('\n')
  .filter((file) => checkedExtensions.test(file) && !file.endsWith('package-lock.json') && existsSync(file))

const failures = []

for (const file of files) {
  const contents = readFileSync(file, 'utf8')

  if (contents.includes('\r\n')) {
    failures.push(`${file}: use LF line endings`)
  }
  if (contents.includes('\u2014')) {
    failures.push(`${file}: replace em dashes`)
  }

  contents.split('\n').forEach((line, index) => {
    if (/\s+$/.test(line)) {
      failures.push(`${file}:${index + 1}: remove trailing whitespace`)
    }
  })
}

if (failures.length > 0) {
  console.error(failures.join('\n'))
  process.exit(1)
}

console.log(`Style checks passed for ${files.length} files.`)
