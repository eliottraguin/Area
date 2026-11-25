# Choosing the Frontend Technologies for the AREA Clients

We are a team of four. The brief asks for a web client that builds workflows and an Android client that monitors and manages them. Both talk to the same server through its OpenAPI contract, and both have to meet the accessibility, localization, and reproducible delivery requirements. See the AREA brief, sections “Clients & Server” and “Deliverables and expectations”, and the scope in #3.

The server is written in Go and stores its data in PostgreSQL, as decided in the [backend choice](backend-choice.md) and the [persistence choice](persistence_choice.md). Neither decision forces a language on the clients, because they only see the OpenAPI contract. Everyone on the team already knew TypeScript, and nobody is recorded as knowing Dart or Kotlin.

## Candidates

The web client is a graph editor where users build workflows, so we looked at the four main web frameworks: React, Vue, Svelte, and Angular. The Android client lists workflows, shows their status, and lets users switch them on or off, so we looked at React Native with Expo, Flutter, Kotlin with Jetpack Compose, and Lynx. We did not keep React Native Web or Flutter Web as options for the web client, because they would give the builder the constraints of a mobile toolkit without a real DOM.

We prototyped React, React Native, and Lynx. We compared Vue, Svelte, Angular, Flutter, and Kotlin only from their documentation, their libraries, and the activity of their repositories, so their scores below rest on less evidence.

## Prototype Comparison

We built the same screen with React and Vite, React Native with Expo, and Lynx (ReactLynx): a list of workflows with their last run status, a switch that enables or disables a workflow through `PATCH /workflows/{id}` with a bearer token, and an EN/FR toggle. All three apps use the same small mock API, described by an OpenAPI file. We then added two things the brief makes important: an OIDC login (authorization code with PKCE) against a mock provider, and, on the web only, a small workflow builder with React Flow showing a trigger, a condition, and two reactions.

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

We looked at the repositories on GitHub on 2026-11-25. All of them received commits that same week.

| Web | React | Vue | Svelte | Angular |
| --- | --- | --- | --- | --- |
| First commit | 2013 | 2018 | 2016 | 2014 |
| Stars | about 253,000 | about 55,000 | about 90,000 | about 102,000 |
| Contributors | about 2,000 | about 670 | about 1,000 | about 2,700 |
| Latest release | v19.3.1 (2026-10-14) | v3.5.46 (2026-11-20) | 5.59.1 (2026-11-20) | v22.3.0 (2026-11-18) |

| Android | React Native | Expo | Flutter | Kotlin (Jetpack Compose) | Lynx |
| --- | --- | --- | --- | --- | --- |
| First commit | 2015 | 2016 | 2015 | not on GitHub | March 2025 |
| Stars | about 127,000 | about 54,000 | about 181,000 | not on GitHub | about 15,000 |
| Contributors | about 4,400 | about 2,300 | about 2,500 | not on GitHub | about 110 |

Jetpack Compose is developed by Google inside the Android Open Source Project, so it has no repository we can compare with the others. Every candidate is actively maintained, and the large frameworks are backed by a company (Meta, Google) or a very large community. Lynx is the exception. It has roughly 20 to 40 times fewer contributors than React Native and Expo and less than two years of history, so we expect to find fewer answers when something breaks.

### Reproduce the Measurements

The prototypes are throwaway and are not in the repository. Each one starts from the official scaffold of its tool.

```sh
npm create vite@latest react-web -- --template react-ts
npx create-expo-app@latest expo-mobile --template blank-typescript
npm create rspeedy@latest lynx-app -- --template react-ts
```

We built them with `npm run build`, `npx expo export --platform android`, and `npm run build` respectively. On the emulator, Expo ran in Expo Go (`npx expo start --android`) and Lynx in the LynxExplorer 4.2.0 APK, opened with `adb shell am start -a android.intent.action.VIEW -d 'lynx://open?url=…'` after an `adb reverse` of the dev server port. The accessibility checks used `adb shell uiautomator dump` with TalkBack enabled, and axe-core through Playwright on the web.

## Scoring

We scored each candidate from 0 to 5 on the criteria of #9, plus maintenance, and for the web client the fit with a graph editor. The scores for React, React Native, and Lynx come from the prototypes. The others come from documentation and repositories, and we capped their accessibility score at 4 because we did not test them. Every score is our judgment and not a measurement. Session security, accessibility, and the graph editor weigh 3, because the brief treats the first two as requirements and the builder is the main job of the web client. Familiarity, effort, OpenAPI, delivery, and maintenance weigh 2. Localization weighs 1, because we only tested a two-entry dictionary.

### Web client

| Criterion | Weight | React | Vue | Svelte | Angular | Why |
| --- | --- | --- | --- | --- | --- | --- |
| Team familiarity | 2 | 3 | 3 | 3 | 3 | We know TypeScript. We did not record who has used any of the four, so they score the same until the team tells us otherwise. |
| Implementation effort | 2 | 4 | 3 | 3 | 2 | We built the screen and the builder with React in a few dozen lines each. Vue and Svelte should be close, but we did not build them. Angular has more concepts to learn, such as dependency injection and its own build tool. |
| OpenAPI integration | 2 | 5 | 5 | 5 | 5 | OpenAPI Generator has `typescript-fetch`, `typescript-axios`, and `typescript-angular` clients. |
| Session security | 3 | 4 | 4 | 4 | 4 | `oidc-client-ts` does not depend on a framework. We only ran it with React, and the real flow is not validated yet. |
| Accessibility | 3 | 5 | 4 | 4 | 4 | axe-core found 0 violations on our React page. We did not test the other three. |
| Localization | 1 | 4 | 4 | 3 | 5 | `i18next` and `vue-i18n` are widely used, Angular ships its own i18n, and Svelte has no official solution. |
| Reproducible delivery | 2 | 4 | 4 | 4 | 4 | All four build a static folder from the command line. |
| Maintenance and ecosystem | 2 | 5 | 4 | 4 | 5 | Contributors and release dates from the snapshot above. |
| Graph editor | 3 | 5 | 4 | 3 | 2 | React Flow gave us a working builder. Svelte Flow comes from the same maintainers (`@xyflow/svelte` 2.0.3, updated 2026-11-23). Vue Flow is maintained by the community (`@vue-flow/core` 1.48.2, last published 2026-01-28). We found no xyflow library for Angular. |
| **Weighted total (/100)** | | **88** | **75** | **77** | **73** | |

The ranking does not depend on our untested accessibility scores. With 5 instead of 4, Svelte reaches 80, Vue 78, and Angular 76, which is still behind React.

### Android client

| Criterion | Weight | React Native (Expo) | Flutter | Kotlin (Compose) | Lynx | Why |
| --- | --- | --- | --- | --- | --- | --- |
| Team familiarity | 2 | 3 | 1 | 1 | 3 | React Native and Lynx use TypeScript and React concepts. Flutter needs Dart and Kotlin is a new language with the Android SDK, and nobody is recorded as knowing either. |
| Implementation effort | 2 | 4 | 3 | 3 | 4 | React Native and Lynx can reuse the generated client and the types of the web client. Flutter and Kotlin need a second language. Lynx's `<list>` needed an explicit size. |
| OpenAPI integration | 2 | 5 | 4 | 4 | 5 | OpenAPI Generator has `dart`, `dart-dio`, and `kotlin` clients, but they would be a second client to maintain next to the TypeScript one. |
| Session security | 3 | 4 | 4 | 4 | 1 | Expo has `expo-secure-store` and `expo-auth-session`, but the Android login did not finish in Expo Go. Flutter has `flutter_secure_storage` (11.2.0, 2026-09-16) and `flutter_appauth` (13.0.0, 2026-10-30). Kotlin uses the Android Keystore directly. We found nothing for Lynx. |
| Accessibility | 3 | 5 | 4 | 4 | 1 | Android listed our switches as real switches in the React Native app. Flutter and Kotlin have their own accessibility APIs, which we did not test. None of the Lynx rows or buttons reached the accessibility tree. |
| Localization | 1 | 4 | 4 | 5 | 3 | React Native has `i18next` and `expo-localization`, Flutter has `intl`, and Kotlin uses the locale and string resources of Android. Lynx documents `i18next`, and we found no device language API. |
| Reproducible delivery | 2 | 4 | 4 | 4 | 2 | React Native, Flutter, and Kotlin build an APK from the command line with Gradle or their own tool, but we only ran the Expo export. A Lynx app needs a native host app, and we only ran it inside LynxExplorer. |
| Maintenance and ecosystem | 2 | 5 | 5 | 5 | 2 | Contributors and activity from the snapshot above. |
| **Weighted total (/85)** | | **73** | **62** | **63** | **41** | |

React Native stays first under the kinder assumptions. With 5 on accessibility for Flutter and Kotlin, they reach 65 and 66, and with 3 on familiarity as well, 69 and 70. If we give Lynx 3 on accessibility and 2 on session security, because LynxExplorer may be the cause of what we saw, it reaches 50.

## Why React and React Native Fit AREA

- **A builder that needs a real DOM.** The workflow builder is a graph editor with nodes, connections, and side panels. React Flow gave us draggable nodes and connections in one component, and the page passed axe-core without changes on our side. The equivalents for Vue and Svelte are younger or smaller, and we found none for Angular. [React Flow](https://reactflow.dev/)
- **One client generated from the OpenAPI contract.** The server is written in Go and publishes an OpenAPI contract. Both clients are TypeScript, so we can generate one typed client from that contract and use it in the web and Android apps. The same goes for the shared types and the workflow validation rules.
- **Session handling on Android.** Expo provides `expo-secure-store`, which keeps the token in the Android Keystore, and `expo-auth-session`, which builds the PKCE request. On the web, `oidc-client-ts` works with a store we replaced by an in-memory one, and we did not find the access token in the browser storage after login. We found no equivalent of these two Expo modules in the Lynx documentation. [expo-secure-store](https://docs.expo.dev/versions/latest/sdk/securestore/) · [expo-auth-session](https://docs.expo.dev/versions/latest/sdk/auth-session/) · [oidc-client-ts](https://github.com/authts/oidc-client-ts)
- **Accessibility on Android.** React Native exposes roles and states, so our switch appears to Android as a checkable switch with its label. Lynx documents `accessibility-element`, `accessibility-label`, and `accessibility-trait`, but the trait only accepts image, button, and text, so a switch with a checked state cannot be expressed. In our test, none of the Lynx rows or buttons reached the accessibility tree. [Lynx accessibility guide](https://lynxjs.org/next/guide/inclusion/accessibility.md)
- **Localization.** `i18next` works on the web, in React Native, and in Lynx, and `expo-localization` gives us the device language on Android. We only tested a two-entry dictionary, so plurals and date formats are still to be checked.
- **Reproducible delivery.** Vite produces a static folder, and Expo can produce the Android project with `expo prebuild` and build it with Gradle, or with EAS. Both can run in a container build stage. We only ran the exports, not a Docker build.

## Security and Dependencies

The mock OIDC provider only exists for the spike. It accepts any redirect URI and signs nothing, so it says nothing about the security of the real flow. What the spike does show is how each client can hold a token: in memory on the web, and in the Keystore on Android through `expo-secure-store`. The Android login is not proven end to end, so we still have to confirm on a development build that the PKCE exchange and the redirect work. We also still have to choose the OIDC library for the real server, check how it is maintained, and validate the `state` and redirect parameters on both clients. [OAuth 2.0 for Native Apps](https://www.rfc-editor.org/rfc/rfc8252.html) · [OAuth 2.0 security recommendations](https://www.rfc-editor.org/rfc/rfc9700.html)

## Decision

We chose React with Vite and TypeScript for the web client, which scored 88 out of 100, and React Native with Expo for the Android client, which scored 73 out of 85.

- **Svelte, Vue, and Angular** came behind on the graph editor, the main job of the web client. Svelte is our fallback for the web at 77, because Svelte Flow comes from the same maintainers as React Flow. Vue Flow is maintained by the community, and we found no xyflow library for Angular.
- **Kotlin and Flutter** scored 63 and 62. They would give us a very good Android client, but they need a second language that nobody on the team knows and a second generated client next to the TypeScript one. We would revisit Kotlin if the Android client needs a platform feature that Expo does not cover.
- **Lynx** scored 41. It builds both clients from one codebase, but the web client is a graph editor that benefits from a real DOM, and we found no secure token storage for it and none of our rows reached the Android accessibility tree. It is also the youngest project of the list.

We accept that we will maintain two UI codebases. We keep the cost down by sharing the generated OpenAPI client, the types, and the workflow logic between them. Screens and components are not shared.
