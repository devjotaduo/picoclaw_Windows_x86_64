import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from '@tanstack/react-router'
import { AppShell } from './components/AppShell'
import { ChatPage } from './routes/chat'
import { AgentsPage } from './routes/agents'
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

const routeTree = rootRoute.addChildren([chatRoute, agentsRoute, modelsRoute, credentialsRoute])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
