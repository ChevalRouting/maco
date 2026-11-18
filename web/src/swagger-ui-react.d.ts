declare module 'swagger-ui-react' {
  import type { ComponentType } from 'react'

  interface SwaggerUIProps {
    url?: string
    spec?: object
    docExpansion?: 'list' | 'full' | 'none'
    deepLinking?: boolean
    tryItOutEnabled?: boolean
  }

  const SwaggerUI: ComponentType<SwaggerUIProps>
  export default SwaggerUI
}
