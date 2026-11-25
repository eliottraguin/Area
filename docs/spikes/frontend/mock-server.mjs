// Minimal mock of openapi.yaml. Run: node mock-server.mjs  (http://localhost:4010)
import http from 'node:http';
const TOKEN = 'spike-token';
const wf = [
  { id: '1', name: 'GitHub issue -> Slack message', enabled: true, lastRun: 'success' },
  { id: '2', name: 'Gmail new mail -> Drive upload', enabled: false, lastRun: 'failed' },
  { id: '3', name: 'Timer 08:00 -> Discord post', enabled: true, lastRun: 'never' },
];
http.createServer((req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'authorization,content-type');
  res.setHeader('Access-Control-Allow-Methods', 'GET,PATCH,OPTIONS');
  if (req.method === 'OPTIONS') return res.writeHead(204).end();
  const send = (c, b) => { res.writeHead(c, { 'content-type': 'application/json' }); res.end(JSON.stringify(b)); };
  if (req.headers.authorization !== `Bearer ${TOKEN}`) return send(401, { error: 'unauthorized' });
  if (req.method === 'GET' && req.url === '/workflows') return send(200, wf);
  const m = req.url.match(/^\/workflows\/(\w+)$/);
  if (req.method === 'PATCH' && m) {
    let body = ''; req.on('data', d => body += d);
    return req.on('end', () => {
      const w = wf.find(x => x.id === m[1]); if (!w) return send(404, {});
      w.enabled = JSON.parse(body).enabled; send(200, w);
    });
  }
  send(404, {});
}).listen(4010, () => console.log('mock on :4010'));
