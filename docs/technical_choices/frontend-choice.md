# Choosing React and React Native for the AREA Clients

We are a team of four. The brief asks for a web client that builds workflows and an Android client that monitors and manages them. Both talk to the same server through its OpenAPI contract, and both have to meet the accessibility, localization, and reproducible delivery requirements. See the AREA brief, sections “Clients & Server” and “Deliverables and expectations”, and the scope in #3. Everyone on the team already knew TypeScript, as noted in the [backend choice](backend-choice.md).

## Prototype Comparison

We compared two approaches. The first uses React with Vite for the web and React Native with Expo for Android. The second uses Lynx (ReactLynx) for both clients. We did not keep React Native Web as a third option, because it would give the web builder the constraints of React Native without a real DOM.

For each approach we built the same screen: a list of workflows with their last run status, a switch that enables or disables a workflow through `PATCH /workflows/{id}` with a bearer token, and an EN/FR toggle. All three apps use the same small mock API, described by an OpenAPI file. We then added two things the brief makes important: an OIDC login (authorization code with PKCE) against a mock provider, and, on the web only, a small workflow builder with React Flow showing a trigger, a condition, and two reactions.

| | React (Vite) | React Native (Expo SDK 58) | Lynx (ReactLynx 0.129) |
| --- | --- | --- | --- |
| Screen code | 39 lines | 65 lines | 48 lines and 5 lines of CSS |
| `npm install` | 3 s, 97 MB | about 15 s, 385 MB | 24 s, 474 MB |
| Build | 1 s | 6 s (`expo export --platform android`) | 5 s, including the type check |
| Output | 221 kB of JavaScript (69 kB gzip), 470 kB with the builder | 1.5 MB Hermes bundle | 99 kB `.lynx.bundle` |

We ran the Android apps on an x86_64 emulator (Android 35, Pixel 6 profile). Expo ran in Expo Go and Lynx in the LynxExplorer 4.2.0 APK from the Lynx releases. The browser checks used Playwright and axe-core.

| Check | React (Vite) | React Native (Expo Go) | Lynx (LynxExplorer) |
| --- | --- | --- | --- |
| The screen shows the API data | Yes | Yes | Yes, once `<list>` has an explicit height. Without it the list stays empty. |
| Toggle and EN/FR switch | Not clicked | Not tapped | Tapped on the emulator, both work |
| axe-core | 0 violations, 3 elements with `role=switch` | Not applicable | Not applicable |
| Android accessibility tree | Not applicable | 3 `android.widget.Switch`, checkable and checked, with labels | Only the title and the native toolbar |
| TalkBack focus | Not applicable | Lands on the app's own elements | Stays on the native back arrow, taps go straight to the app |
| OIDC with PKCE | Works: login, user name shown, list loaded with the token, token absent from local and session storage | Partial, see below | Not attempted |
| Workflow builder | Dragging a connection works (2 to 3 edges), nodes are focusable, axe-core reports 0 violations | Not applicable | Not applicable |
| Web target | Native | Not used | The same source builds a web bundle (97.6 kB), but we did not get it to render in a browser |

On Android, the Expo login opens a Chrome Custom Tab on the provider, and the redirect comes back to Expo Go. Expo Go then restarts the app, so the code exchange does not complete. A development build would be needed to finish the test.

Lynx needed its own tags (`view`, `text`, `list`), React Native its own components, and React the usual HTML elements. Writing the screen took about the same effort in the three cases. Lynx is the smallest bundle, but the runtime lives in the host app, so the sizes cannot be compared directly with Hermes or with web JavaScript.

### Maintenance

We looked at the three repositories on GitHub on 2026-11-24.

| | React Native | Expo | Lynx |
| --- | --- | --- | --- |
| First commit | 2015 | 2016 | March 2025 |
| Stars | about 127,000 | about 54,000 | about 15,000 |
| Contributors | about 4,400 | about 2,300 | about 110 |
| Latest release | v0.88.1 (2026-10-27) | SDK templates updated 2026-11-17 | 4.2.0 (2026-10-27) |

All three are active. React Native and Expo have roughly 20 to 40 times more contributors than Lynx, and years more production use, so we expect to find answers faster when something breaks.

### Reproduce the Measurements

The prototypes are throwaway and are not in the repository. Each one starts from the official scaffold of its tool.

```sh
npm create vite@latest react-web -- --template react-ts
npx create-expo-app@latest expo-mobile --template blank-typescript
npm create rspeedy@latest lynx-app -- --template react-ts
```

We built them with `npm run build`, `npx expo export --platform android`, and `npm run build` respectively. On the emulator, Expo ran in Expo Go (`npx expo start --android`) and Lynx in the LynxExplorer 4.2.0 APK, opened with `adb shell am start -a android.intent.action.VIEW -d 'lynx://open?url=…'` after an `adb reverse` of the dev server port. The accessibility checks used `adb shell uiautomator dump` with TalkBack enabled, and axe-core through Playwright on the web.
