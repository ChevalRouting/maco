import SwaggerUI from 'swagger-ui-react'
import 'swagger-ui-react/swagger-ui.css'

export default function ApiDocsPanel() {
  return (
    <div className="api-reference overflow-hidden rounded-xl">
      <SwaggerUI url="/api/docs/openapi.json" docExpansion="none" deepLinking />
    </div>
  )
}
