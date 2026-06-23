/// <reference types="vite/client" />

interface ImportMetaEnv {
  /**
   * When "true", the app serves every API call from the in-browser mock
   * (src/api/mock) and never contacts the backend. See src/api/mock/README.md.
   */
  readonly VITE_USE_MOCK_API?: string;
  /** Dev-proxy target for /api (ignored when mock mode is on). */
  readonly VITE_API_TARGET?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
