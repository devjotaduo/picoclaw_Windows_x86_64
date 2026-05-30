import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from '@tanstack/react-router'
import { AppShell } from './components/AppShell'
import { ChatPage } from './routes/chat'
import { AgentsPage } from './routes/agents'
import { AgentChatPage } from './routes/agent-chat'
import { WhatsAppPage } from './routes/whatsapp'
import { ModelsPage } from './routes/models'
import { CredentialsPage } from './routes/credentials'

const rootRoute = createRootRoute({
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
})

const chatRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: ChatPage,
})

const agentsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/agents',
  component: AgentsPage,
})

const agentChatRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/a/$name',
  component: AgentChatPage,
})

const whatsappRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/whatsapp',
  component: WhatsAppPage,
})

const modelsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/models',
  component: ModelsPage,
})

const credentialsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/credentials',
  component: CredentialsPage,
})

const routeTree = rootRoute.addChildren([
  chatRoute,
  agentsRoute,
  agentChatRoute,
  whatsappRoute,
  modelsRoute,
  credentialsRoute,
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
