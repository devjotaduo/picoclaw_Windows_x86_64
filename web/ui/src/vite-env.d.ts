/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Backend origin for cross-origin deploys (e.g. Vercel → Railway). Empty in
  // local/embedded builds, where requests are same-origin.
  readonly VITE_API_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
