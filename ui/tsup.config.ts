import { defineConfig } from 'tsup'
import { fileURLToPath } from 'node:url'

const src = fileURLToPath(new URL('./src', import.meta.url))

export default defineConfig({
  entry: {
    index: 'src/index.ts',
    charts: 'src/charts.ts',
    terminal: 'src/terminal.ts',
  },
  format: ['esm'],
  dts: true,
  sourcemap: true,
  clean: true,
  treeshake: true,
  external: [
    'react',
    'react-dom',
    'react/jsx-runtime',
    'react-router-dom',
    'recharts',
    '@xterm/xterm',
    '@xterm/addon-fit',
    '@xterm/xterm/css/xterm.css',
  ],
  esbuildOptions(options) {
    options.alias = { '@': src }
  },
  onSuccess: 'cp src/styles/tokens.css dist/styles.css',
})
