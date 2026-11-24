# Spike Go — Bootstrap AREA

**Timebox prévue** — 25 minutes, comme pour Rust et TypeScript. Une seule route HTTP suffit pour comparer la prise en main et les étapes nécessaires pour lancer chaque serveur.

## Résultat

J’ai atteint la cible du spike Go en **12 minutes**.

- J’ai créé un serveur HTTP en Go avec `net/http`, la bibliothèque standard.
- Le serveur lit le port avec `os.Getenv("PORT")`. Je peux le lancer sur le port demandé avec `PORT=8080 go run .`.
- `GET /ping` renvoie exactement `{"status":"ok"}`, sans saut de ligne.
- J’ai aussi ajouté `/hello`, qui affiche le chemin demandé, et une page d’accueil blanche avec le titre « Mon projet AREA ».

## Difficultés rencontrées

La recherche dans la documentation Go est l’étape qui m’a pris le plus de temps.

J’ai ajusté le format de la réponse de `/ping` pour respecter le JSON exact. J’ai aussi identifié le `%` affiché par zsh quand la réponse se termine sans saut de ligne.

## Pour aller plus loin

Je limiterais `/ping` à la méthode `GET` et j’ajouterais un test qui vérifie le code HTTP et le corps exact. À ce stade, `go test ./...` compile le programme, mais aucun test automatisé n’est présent.

## Reproduire

Depuis `bootstrap/Go`, lancer `PORT=8080 go run .`, puis exécuter `curl -w '\n' http://localhost:8080/ping` dans un second terminal. L’option `-w` ajoute un saut de ligne à l’affichage.
