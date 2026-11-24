# Choix de Go pour le backend AREA

Nous sommes un groupe de quatre. Notre backend doit fournir une API commune aux clients web et Android, détecter des événements même quand les clients sont fermés et exécuter les workflows associés. Le sujet nous demande au moins cinq services et dix Actions et REActions au total. Au moins la moitié des services doivent demander une autorisation OAuth2. Voir le sujet AREA, sections « Hooks », « Clients & Server » et « Deliverables and expectations ».

## Comparaison des prototypes

Nous avons comparé Go, Rust et TypeScript à partir de trois essais limités à 25 minutes chacun. Nous avons utilisé la même cible pour les trois. Le serveur devait lire son port depuis `PORT`, pouvoir écouter sur `8080` et renvoyer exactement `{"status":"ok"}` sur `GET /ping`.

| Technologie | Dépendances HTTP du prototype | Premier succès | Étape la plus longue de l’essai |
| --- | --- | --- | --- |
| Go | `net/http`, fourni par la bibliothèque standard | 12 min | Chercher dans la documentation Go. |
| Rust | Axum et Tokio | 20 min | Comprendre comment mettre en place le serveur et trouver la démarche. |
| TypeScript | Express sur Node.js | 5 min | Se remémorer les connaissances déjà acquises dans le langage. |

Les trois prototypes ont atteint la cible. Les temps mesurent la mise en place de cette route et le réglage du port pendant le spike. TypeScript était déjà familier à toute l’équipe, ce qui rendait ce choix plus prévisible pour démarrer. [Comparaison des essais](spikes/DISTANCE.md) · [Code Go](spikes/Go/server.go) · [Code Rust](spikes/Rust/src/main.rs) · [Code TypeScript](spikes/Typescript/main.ts)

### Reproduire le prototype Go

Depuis la racine du dépôt, lancer le serveur sur un port `8080` libre.

```sh
cd docs/spikes/Go
PORT=8080 go run .
```

Dans un second terminal, `curl -fsS http://localhost:8080/ping` doit renvoyer `{"status":"ok"}`.

## Pourquoi Go convient au projet AREA

- **Une API commune aux deux clients.** Nous pouvons construire les routes avec `net/http`, y compris celle qui expose le catalogue des Services, Actions et REActions. Avec `httptest`, nous pouvons envoyer des requêtes aux handlers et vérifier leurs réponses. Ces outils nous donnent une base pour écrire le test demandé par le sujet, qui doit échouer si l’implémentation s’écarte du contrat OpenAPI, y compris sur les réponses d’erreur. [net/http](https://pkg.go.dev/net/http) · [httptest](https://pkg.go.dev/net/http/httptest)
- **Des hooks autonomes.** Nous pouvons utiliser des goroutines pour faire tourner la détection des événements en parallèle du serveur HTTP et `context` pour arrêter proprement ces tâches. L’ordre des réactions reste géré par le moteur à l’intérieur de chaque workflow. Nous devons aussi définir comment identifier et mémoriser les événements déjà traités, comme le demande le sujet. [Goroutines](https://go.dev/doc/effective_go#goroutines) · [context](https://pkg.go.dev/context)
- **Des appels aux services tiers.** Le client de `net/http` permet de fixer des délais pour nos appels sortants. Si le service appelé par une réaction tarde à répondre, nous pouvons faire remonter l’échec au moteur. Cette brique peut être partagée entre plusieurs intégrations, même si chaque service conserve ses propres paramètres et autorisations. [Client HTTP de Go](https://pkg.go.dev/net/http#Client)
- **Un modèle explicite pour les workflows.** Les structures Go nous permettent de représenter les déclencheurs, les réactions et les conditions avec des données typées. Nous pouvons écrire des validations pour appliquer les règles du sujet au moment où un workflow est enregistré, par exemple vérifier qu’il comporte au plus une condition et que chaque référence vise une donnée déclarée par un nœud précédent.
- **Un backend à intégrer dans Docker Compose.** `go build` produit un exécutable pour le serveur. Nous pouvons compiler cet exécutable dans une étape de construction du conteneur, puis le lancer depuis l’image finale. C’est une base pour la construction reproductible demandée par le sujet. [Commande go build](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)

## Sécurité et dépendances

Pour cette route, Go utilise uniquement la bibliothèque standard, alors que les deux autres prototypes ajoutent des dépendances HTTP externes. La sécurité OAuth2 dépendra, pour les trois options, de la bibliothèque choisie, de sa maintenance et de son utilisation. Nous prévoyons l’intégration OAuth2 pour les services tiers demandés par le sujet. Le choix de la bibliothèque reste à faire.

Pour AREA, les vérifications à prévoir concernent notamment la validation du retour OAuth2, la limitation des permissions demandées et la protection des jetons que le serveur conserve pour les hooks. Un premier essai possible serait de connecter un vrai service, puis de vérifier qu’un retour OAuth2 invalide ne permet pas de connecter le compte. Le spike actuel couvre le démarrage du serveur et le comportement de `/ping`. [Recommandations OAuth 2.0](https://www.rfc-editor.org/rfc/rfc9700.html)

## Décision

Nous avons choisi **Go** parce que nous voulions découvrir une nouvelle technologie en groupe. Nous avions tous déjà fait du TypeScript, qui nous aurait permis d’avancer plus vite au départ. Rust offrait l’occasion d’explorer ses possibilités d’optimisation, et son essai a demandé plus de temps pour comprendre comment mettre en place le serveur.

Pour nous, Go représente un entre-deux. Son apprentissage nous permet de découvrir une nouvelle technologie, et la première route a fonctionné en 12 minutes avec la bibliothèque standard. Nous acceptons que la prise en main demande plus de temps qu’avec TypeScript.
