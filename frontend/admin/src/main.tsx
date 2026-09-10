import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { configureApi } from '@common/api/client'
import './index.css'
import { App } from './app/App'

configureApi({ refreshPath: '/openapi/admin/auth/refresh' })

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
