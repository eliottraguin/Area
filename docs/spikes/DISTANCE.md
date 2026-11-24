# Distance parcourue — Spikes AREA

Les trois essais ont eu une **timebox de 25 minutes**. Ce créneau permet de tenter une route HTTP minimale et de relever les points qui ralentissent la prise en main de chaque stack.

## Méthode de comparaison

La distance parcourue correspond au nombre de critères du sujet implémentés, sur trois. Chaque critère vaut un point.

1. Le serveur peut écouter sur le port `8080`.
2. `GET /ping` renvoie exactement `{"status":"ok"}`, sans saut de ligne.
3. Le port est configurable par la variable d’environnement `PORT`.

Ces critères donnent la même cible aux trois essais. Le temps jusqu’au premier succès et l’étape la plus longue complètent le score en montrant l’effort nécessaire pour y arriver. Les routes supplémentaires restent hors du score.

## Comparaison

| Critère | Go | Rust | TypeScript |
| --- | --- | --- | --- |
| Stack HTTP | `net/http` (bibliothèque standard) | Axum et Tokio | Express et Node.js |
| Lancement sur `8080` | `PORT=8080 go run .` | `PORT=8080 cargo run --locked` | `PORT=8080 node main.ts` |
| Corps exact de `GET /ping` | Implémenté | Implémenté | Implémenté |
| Lecture du port | `os.Getenv("PORT")` | `std::env::var("PORT")`, défaut `8080` | `process.env.PORT`, défaut `3000` |
| **Distance parcourue** | **3/3** | **3/3** | **3/3** |
| **Temps jusqu’au premier succès** | **12 min** | **20 min** | **5 min** |
| Étape la plus longue | Chercher dans la documentation Go. | Comprendre comment faire et trouver la démarche pour monter le serveur. | Se remémorer les connaissances déjà acquises en TypeScript. |
| Compte rendu | [SPIKE.md](Go/SPIKE.md) | [SPIKE.md](Rust/SPIKE.md) | [SPIKE.md](Typescript/SPIKE.md) |

Les réponses Rust et TypeScript ont été vérifiées avec `curl` sur des ports libres fournis via `PORT`, car `8080` était occupé. Le handler Go a été vérifié avec `httptest`, qui a confirmé le statut `200` et le corps exact.

## Bilan

Les trois prototypes ont atteint la cible dans les 25 minutes. TypeScript a été le plus rapide à mettre en place pendant ces essais, avec un premier succès en 5 minutes, devant Go en 12 minutes et Rust en 20 minutes. La connaissance préalable de TypeScript a facilité le démarrage. En Go, la recherche dans la documentation a pris le plus de temps. En Rust, c’était la compréhension de la démarche pour mettre le serveur en place.
