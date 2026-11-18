import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: '../pkg/api/docs/swagger.json',
  output: 'src/api/generated',
  plugins: [
    { name: '@hey-api/client-fetch', throwOnError: true },
    { name: '@hey-api/sdk', operations: { strategy: 'single', containerName: 'DefaultApi' } },
    '@hey-api/typescript',
  ],
})
