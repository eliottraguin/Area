# Spike Rust — Bootstrap AREA

**Timebox prévue** — 25 minutes, comme pour les deux autres essais. Une route HTTP minimale suffit pour comparer la prise en main, le démarrage du serveur et la configuration du port.

## Résultat

J’ai atteint la cible du spike Rust en **20 minutes**.

- J’ai mis en place un serveur HTTP avec Axum pour le routage et Tokio pour l’exécution asynchrone.
- Le serveur lit `PORT` avec `std::env::var`. La valeur par défaut est `8080`.
- `GET /ping` renvoie exactement `{"status":"ok"}`, sans saut de ligne. J’ai aussi ajouté une route `GET /hello`.

## Difficultés rencontrées

L’étape qui m’a pris le plus de temps a été de comprendre comment faire et de trouver la démarche pour mettre en place le serveur en Rust.

## Pour aller plus loin

J’ajouterais un test HTTP automatisé pour vérifier le statut et le corps exact de `/ping`, puis je testerais un appel à une API externe depuis un handler asynchrone.

## Reproduire

Depuis `bootstrap/Rust`, lancer `PORT=8080 cargo run --locked`, puis exécuter `curl http://localhost:8080/ping` dans un second terminal.

Le prototype a été compilé avec `cargo build --locked --offline`. Un essai HTTP sur un autre port libre a confirmé que le serveur utilise `PORT` et que le corps de `/ping` correspond exactement au résultat attendu.
