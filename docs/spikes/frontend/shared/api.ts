// Shared by the three spikes (copied, not imported, to keep each app standalone).
export type Workflow = { id: string; name: string; enabled: boolean; lastRun: 'success' | 'failed' | 'never' };
const BASE = 'http://localhost:4010';
const headers = (t: string) => ({ authorization: `Bearer ${t}`, 'content-type': 'application/json' });
export async function listWorkflows(token: string): Promise<Workflow[]> {
  const r = await fetch(`${BASE}/workflows`, { headers: headers(token) });
  if (!r.ok) throw new Error(String(r.status));
  return r.json();
}
export async function setWorkflowEnabled(token: string, id: string, enabled: boolean): Promise<Workflow> {
  const r = await fetch(`${BASE}/workflows/${id}`, { method: 'PATCH', headers: headers(token), body: JSON.stringify({ enabled }) });
  if (!r.ok) throw new Error(String(r.status));
  return r.json();
}
