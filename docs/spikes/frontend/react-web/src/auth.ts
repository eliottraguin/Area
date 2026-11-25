import { UserManager, WebStorageStateStore } from 'oidc-client-ts';
// Authorization code + PKCE (oidc-client-ts does PKCE by default). Tokens live in memory only.
export const auth = new UserManager({
  authority: 'http://localhost:4011',
  client_id: 'area-web',
  redirect_uri: window.location.origin + '/',
  response_type: 'code',
  scope: 'openid profile',
  userStore: new WebStorageStateStore({ store: new (class { // in-memory store, nothing persisted in the browser
    m = new Map<string, string>(); length = 0;
    key(i: number) { return [...this.m.keys()][i] ?? null; }
    getItem(k: string) { return this.m.get(k) ?? null; }
    setItem(k: string, v: string) { this.m.set(k, v); this.length = this.m.size; }
    removeItem(k: string) { this.m.delete(k); this.length = this.m.size; }
    clear() { this.m.clear(); this.length = 0; }
  })() as unknown as Storage }),
});
