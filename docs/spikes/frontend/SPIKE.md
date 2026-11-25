# Frontend Spike for the AREA Clients

**Goal** — Build the same screen with React and React Native, and with Lynx, so we can compare them on the points the brief makes important for the web and Android clients.

## Result

I built one screen in three apps. It lists workflows with their last run status, switches a workflow on or off through `PATCH /workflows/{id}` with a bearer token, and toggles between English and French. The three apps use the same mock API, described by `openapi.yaml`.

- `react-web` is React with Vite. I added a small workflow builder with React Flow and an OIDC login with `oidc-client-ts`.
- `expo-mobile` is React Native with Expo. It keeps the token in `expo-secure-store` and starts the login with `expo-auth-session`.
- `lynx-app` is ReactLynx. It builds a `.lynx.bundle` for Android and a `.web.bundle` for the web.

All three build. I ran the two Android apps on an emulator and the web app in Chromium with Playwright and axe-core. The numbers and the comparison are in [the frontend choice](../../technical_choices/frontend-choice.md).

## Challenges

- Lynx's `<list>` stayed empty until I gave it an explicit height.
- With TalkBack on, the Lynx rows and buttons did not appear in the accessibility tree.
- Expo Go restarts the app when the login redirects back, so the Android OIDC login did not finish.
- I could not get the Lynx web bundle to render. A host page with `@lynx-js/web-core` hit WebAssembly errors with Vite.

## Next Steps

I would try the Android login in a development build with `npx expo run:android`. I would also try Lynx inside a small host app that enables accessibility, to see whether the Lynx result comes from LynxExplorer or from Lynx itself.

## Reproduce the Spike

Start the mock servers from this folder.

```sh
node mock-server.mjs   # API on :4010, token "spike-token"
node oidc-mock.mjs     # OIDC provider on :4011 (code with PKCE), only for the spike
```

Build each app from its folder.

```sh
cd react-web && npm install && npm run build
cd expo-mobile && npm install && npx tsc --noEmit && npx expo export --platform android
cd lynx-app && npm install && npm run build
```

To run the Android apps, start an emulator and forward the ports.

```sh
for p in 4010 4011 8081 3000; do adb reverse tcp:$p tcp:$p; done
```

- Expo: `cd expo-mobile && npx expo start --android` opens the app in Expo Go.
- Lynx: install `LynxExplorer-noasan-release.apk` from the [Lynx 4.2.0 release](https://github.com/lynx-family/lynx/releases/tag/4.2.0), run `cd lynx-app && npm run dev`, then
  `adb shell am start -a android.intent.action.VIEW -d 'lynx://open?url=http%3A%2F%2Flocalhost%3A3000%2Fmain.lynx.bundle'`.

The browser checks are in `check/`. Run `npm install && npx playwright install chromium` there, then `node check.mjs <url> <screenshot.png>` for axe-core, `node builder-check.mjs <url>` to drag a connection in the builder, and `node oidc-check.mjs <url>` for the login. Use the address of `npm run dev` in `react-web` (port 5173).
