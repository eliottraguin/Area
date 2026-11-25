import { useEffect, useState } from 'react';
import { Pressable, SafeAreaView, FlatList, Text, View, StyleSheet } from 'react-native';
import * as SecureStore from 'expo-secure-store';
import * as AuthSession from 'expo-auth-session';
import * as WebBrowser from 'expo-web-browser';
import { listWorkflows, setWorkflowEnabled, type Workflow } from './src/api';
import { messages, type Lang } from './src/i18n';

WebBrowser.maybeCompleteAuthSession();
const ISSUER = 'http://localhost:4011';

export default function App() {
  const [lang, setLang] = useState<Lang>('en');
  const [items, setItems] = useState<Workflow[]>([]);
  const [failed, setFailed] = useState(false);
  const [token, setToken] = useState('');
  const t = messages[lang];

  const discovery = AuthSession.useAutoDiscovery(ISSUER);
  const redirectUri = AuthSession.makeRedirectUri();
  // Authorization code + PKCE (usePKCE defaults to true).
  const [request, response, promptAsync] = AuthSession.useAuthRequest(
    { clientId: 'area-mobile', scopes: ['openid', 'profile'], redirectUri, usePKCE: true },
    discovery,
  );

  useEffect(() => {
    SecureStore.getItemAsync('access_token').then((tk) => tk && setToken(tk));
  }, []);

  useEffect(() => {
    if (response?.type !== 'success' || !request || !discovery) return;
    AuthSession.exchangeCodeAsync(
      { clientId: 'area-mobile', code: response.params.code, redirectUri, extraParams: { code_verifier: request.codeVerifier ?? '' } },
      discovery,
    )
      .then(async (r) => {
        // Session handling: token kept in Android Keystore via SecureStore, not in plain storage.
        await SecureStore.setItemAsync('access_token', r.accessToken);
        setToken(r.accessToken);
      })
      .catch(() => setFailed(true));
  }, [response]);

  useEffect(() => {
    if (token) listWorkflows(token).then(setItems).catch(() => setFailed(true));
  }, [token]);

  const toggle = async (w: Workflow) => {
    const u = await setWorkflowEnabled(token, w.id, !w.enabled);
    setItems((xs) => xs.map((x) => (x.id === u.id ? u : x)));
  };

  return (
    <SafeAreaView style={s.root}>
      <Text accessibilityRole="header" style={s.h1}>{t.title}</Text>
      <Pressable accessibilityRole="button" onPress={() => setLang(lang === 'en' ? 'fr' : 'en')}>
        <Text>{lang === 'en' ? 'FR' : 'EN'}</Text>
      </Pressable>
      {!token && (
        <Pressable accessibilityRole="button" disabled={!request} onPress={() => promptAsync()}>
          <Text>Sign in</Text>
        </Pressable>
      )}
      {failed && <Text accessibilityRole="alert">{t.error}</Text>}
      <FlatList
        data={items}
        keyExtractor={(w) => w.id}
        renderItem={({ item: w }) => (
          <View style={s.row}>
            <View style={{ flex: 1 }}>
              <Text>{w.name}</Text>
              <Text style={s.small}>{t[w.lastRun]}</Text>
            </View>
            <Pressable
              accessibilityRole="switch"
              accessibilityState={{ checked: w.enabled }}
              accessibilityLabel={`${w.name}: ${w.enabled ? t.on : t.off}`}
              onPress={() => toggle(w)}
            >
              <Text>{w.enabled ? t.on : t.off}</Text>
            </Pressable>
          </View>
        )}
      />
    </SafeAreaView>
  );
}

const s = StyleSheet.create({
  root: { flex: 1, padding: 16, paddingTop: 48 },
  h1: { fontSize: 24, fontWeight: '600' },
  row: { flexDirection: 'row', paddingVertical: 12, alignItems: 'center' },
  small: { color: '#555', fontSize: 12 },
});
