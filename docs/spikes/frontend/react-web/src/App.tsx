import { useEffect, useState } from 'react';
import { listWorkflows, setWorkflowEnabled, type Workflow } from './api';
import { messages, type Lang } from './i18n';
import Builder from './Builder';

import { auth } from './auth';

export default function App() {
  const [lang, setLang] = useState<Lang>('en');
  const [items, setItems] = useState<Workflow[]>([]);
  const [failed, setFailed] = useState(false);
  const [token, setToken] = useState('');
  const [name, setName] = useState('');
  const t = messages[lang];

  useEffect(() => {
    (async () => {
      if (new URLSearchParams(window.location.search).has('code')) {
        const u = await auth.signinRedirectCallback();
        window.history.replaceState({}, '', '/');
        setToken(u.access_token); setName(String(u.profile.name ?? ''));
      }
    })().catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    if (token) listWorkflows(token).then(setItems).catch(() => setFailed(true));
  }, [token]);

  const toggle = async (w: Workflow) => {
    const u = await setWorkflowEnabled(token, w.id, !w.enabled);
    setItems((xs) => xs.map((x) => (x.id === u.id ? u : x)));
  };

  return (
    <main lang={lang}>
      <h1>{t.title}</h1>
      <button onClick={() => setLang(lang === 'en' ? 'fr' : 'en')}>{lang === 'en' ? 'FR' : 'EN'}</button>
      {token ? <p data-testid="user">{name}</p> : <button onClick={() => auth.signinRedirect()}>Sign in</button>}
      {failed && <p role="alert">{t.error}</p>}
      <ul>
        {items.map((w) => (
          <li key={w.id}>
            <span>{w.name}</span> <small>{t[w.lastRun]}</small>
            <button role="switch" aria-checked={w.enabled} aria-label={`${w.name}: ${w.enabled ? t.on : t.off}`} onClick={() => toggle(w)}>
              {w.enabled ? t.on : t.off}
            </button>
          </li>
        ))}
      </ul>
      <Builder />
    </main>
  );
}
