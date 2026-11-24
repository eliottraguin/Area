# Spike TypeScript — Bootstrap AREA

**Timebox prévue** — 25 minutes, comme pour les deux autres essais. Une route HTTP minimale suffit pour comparer la prise en main, le démarrage du serveur et la configuration du port.

## Résultat

J’ai atteint la cible du spike TypeScript en **5 minutes**.

- J’ai créé un serveur HTTP avec Express et ajouté des types `Request` et `Response` aux handlers.
- Le serveur lit `process.env.PORT`. La valeur par défaut est `3000`, donc je le lance avec `PORT=8080` pour l’exercice.
- `GET /ping` renvoie exactement `{"status":"ok"}`, sans saut de ligne. La route `GET /` affiche `Hello, World!`.

## Prise en main

Je connaissais déjà TypeScript, donc la prise en main a été rapide. L’étape qui m’a pris le plus de temps a été de me remettre en tête ce que je connaissais déjà du langage.

## Pour aller plus loin

J’ajouterais une commande de lancement dans `package.json` et un test HTTP automatisé pour vérifier le statut et le corps exact de `/ping`.

## Reproduire

Depuis `docs/spikes/Typescript`, installer les dépendances avec `npm ci`, puis lancer `PORT=8080 node main.ts` avec Node.js 24. Dans un second terminal, exécuter `curl http://localhost:8080/ping`.

Un essai HTTP sur un autre port libre a confirmé que le serveur utilise `PORT` et que le corps de `/ping` correspond exactement au résultat attendu.
