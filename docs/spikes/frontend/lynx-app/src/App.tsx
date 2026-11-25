import { useEffect, useState } from '@lynx-js/react';
import { listWorkflows, setWorkflowEnabled, type Workflow } from './api.js';
import { messages, type Lang } from './i18n.js';
import './App.css';

const TOKEN = 'spike-token';

export function App() {
  const [lang, setLang] = useState<Lang>('en');
  const [items, setItems] = useState<Workflow[]>([]);
  const [failed, setFailed] = useState(false);
  const t = messages[lang];

  useEffect(() => {
    listWorkflows(TOKEN).then(setItems).catch(() => setFailed(true));
  }, []);

  const toggle = async (w: Workflow) => {
    const u = await setWorkflowEnabled(TOKEN, w.id, !w.enabled);
    setItems((xs) => xs.map((x) => (x.id === u.id ? u : x)));
  };

  return (
    <view className="root">
      <text className="h1">{t.title}</text>
      <view bindtap={() => setLang(lang === 'en' ? 'fr' : 'en')}>
        <text>{lang === 'en' ? 'FR' : 'EN'}</text>
      </view>
      {failed && <text>{t.error}</text>}
      <list className="list" scroll-orientation="vertical" list-type="single" span-count={1}>
        {items.map((w) => (
          <list-item item-key={w.id} key={w.id}>
            <view className="row">
              <view className="grow">
                <text>{w.name}</text>
                <text className="small">{t[w.lastRun]}</text>
              </view>
              {/* Lynx accessibility-trait only supports image, button, text: no switch role, no checked state (see docs) */}
              <view accessibility-element={true} accessibility-trait="button" accessibility-label={`${w.name}: ${w.enabled ? t.on : t.off}`} bindtap={() => toggle(w)}>
                <text>{w.enabled ? t.on : t.off}</text>
              </view>
            </view>
          </list-item>
        ))}
      </list>
    </view>
  );
}
