import stylistic from '@stylistic/eslint-plugin'
import parser from '@typescript-eslint/parser'
import stylecheck from './lint/rules.mjs'

export default [
  { ignores: ['src/api/generated/**', 'node_modules/**', 'dist/**'] },
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      parser,
      ecmaVersion: 'latest',
      sourceType: 'module',
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    linterOptions: { reportUnusedDisableDirectives: 'error' },
    plugins: { stylecheck, stylistic },
    rules: {
      'stylecheck/no-comments': 'error',
      'stylecheck/shared-controls': 'error',
      'stylecheck/semantic-palette': 'error',
      'stylecheck/neutral-table-actions': 'error',
      'stylecheck/named-handlers': 'error',
      'stylecheck/named-types': 'error',
      'stylecheck/accessible-icon-actions': 'error',
      'stylistic/no-trailing-spaces': 'error',
      'stylistic/eol-last': ['error', 'always'],
      'stylistic/semi': ['error', 'never'],
      'stylistic/quotes': ['error', 'single', { avoidEscape: true }],
    },
  },
]
