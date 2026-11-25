// Minimal OIDC provider for the spike (authorization code + PKCE S256). Not secure, never reuse.
// Run: node oidc-mock.mjs  (http://localhost:4011). Issues access_token "spike-token" accepted by mock-server.mjs.
import http from 'node:http';
import crypto from 'node:crypto';
const ISS = 'http://localhost:4011';
const codes = new Map();
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url');
http.createServer((req, res) => {
  const u = new URL(req.url, ISS);
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'content-type,authorization');
  if (req.method === 'OPTIONS') return res.writeHead(204).end();
  const json = (c, b) => { res.writeHead(c, { 'content-type': 'application/json' }); res.end(JSON.stringify(b)); };
  if (u.pathname === '/.well-known/openid-configuration') return json(200, {
    issuer: ISS, authorization_endpoint: `${ISS}/authorize`, token_endpoint: `${ISS}/token`,
    userinfo_endpoint: `${ISS}/userinfo`, jwks_uri: `${ISS}/jwks`, response_types_supported: ['code'],
    code_challenge_methods_supported: ['S256'], subject_types_supported: ['public'], id_token_signing_alg_values_supported: ['none'],
  });
  if (u.pathname === '/jwks') return json(200, { keys: [] });
  if (u.pathname === '/authorize') { // auto-approves the "user"
    const q = u.searchParams;
    if (q.get('code_challenge_method') !== 'S256' || !q.get('code_challenge')) return json(400, { error: 'pkce_required' });
    const code = crypto.randomUUID();
    codes.set(code, { challenge: q.get('code_challenge'), nonce: q.get('nonce'), client: q.get('client_id'), redirect: q.get('redirect_uri') });
    const back = new URL(q.get('redirect_uri')); back.searchParams.set('code', code); back.searchParams.set('state', q.get('state'));
    if (/^https?:$/.test(back.protocol)) { res.writeHead(302, { location: back.toString() }); return res.end(); }
    // Custom-scheme redirects (mobile) need a user gesture in Chrome, so show an approve page like a real provider.
    res.writeHead(200, { 'content-type': 'text/html' });
    return res.end(`<!doctype html><meta name=viewport content="width=device-width"><title>Approve</title><h1>Sign in to AREA</h1><p><a id=ok href="${back.toString().replace(/"/g, '&quot;')}" style="font-size:2em;padding:1em;display:inline-block;background:#26a;color:#fff">Approve</a></p>`);
  }
  if (u.pathname === '/token' && req.method === 'POST') {
    let body = ''; req.on('data', (d) => (body += d));
    return req.on('end', () => {
      const f = new URLSearchParams(body); const c = codes.get(f.get('code')); codes.delete(f.get('code'));
      if (!c) return json(400, { error: 'invalid_grant' });
      const ok = crypto.createHash('sha256').update(f.get('code_verifier') ?? '').digest('base64url') === c.challenge;
      if (!ok) return json(400, { error: 'invalid_grant', error_description: 'PKCE mismatch' });
      const now = Math.floor(Date.now() / 1000);
      const id_token = `${b64({ alg: 'none', typ: 'JWT' })}.${b64({ iss: ISS, sub: 'user-1', aud: c.client, exp: now + 3600, iat: now, nonce: c.nonce, name: 'Spike User' })}.`;
      json(200, { access_token: 'spike-token', token_type: 'Bearer', expires_in: 3600, id_token });
    });
  }
  if (u.pathname === '/userinfo') return json(200, { sub: 'user-1', name: 'Spike User' });
  json(404, {});
}).listen(4011, () => console.log('oidc mock on :4011'));
