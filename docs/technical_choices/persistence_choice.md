## 1. Besoin

AREA stocke et manipule :

| Donnée | Nature | Contraintes |
|---|---|---|
| Utilisateurs, rôles | relationnelle | unicité email, suppression de compte (RGPD) |
| Services connectés + tokens OAuth | relationnelle + secret | chiffrement au repos, rotation |
| Workflows (AREA) et nodes | graphe / document | structure variable selon le type de node, versionnée |
| Événements détectés par les hooks | écriture fréquente | déduplication : un même événement ne doit jamais être traité deux fois |
| Runs de workflow (historique) | append-only | entrées/sorties de chaque node, durée, statut, métriques |
| Travaux en attente / retry / dead letter | file | pas de perte, rejouable, effet de bord non dupliqué |

Exigences transverses tirées du sujet : hooks autonomes (sans client ouvert), backups horodatés hors du conteneur de la base, health endpoint distinguant la base, chiffrement de ce que révèle un dump, inventaire des données personnelles.

## 2. Critères d'évaluation

| # | Critère | Poids | Pourquoi |
|---|---|---|---|
| C1 | Intégrité / transactions (ACID) | 3 | un run ne doit pas être à moitié enregistré |
| C2 | Concurrence d'écriture | 3 | hooks + workers + API écrivent en parallèle |
| C3 | Déduplication / idempotence | 3 | exigence centrale des hooks et du retry |
| C4 | Modélisation (relationnel + semi-structuré) | 2 | workflows aux nodes hétérogènes |
| C5 | Sécurité (chiffrement, droits, surface) | 3 | tokens OAuth, dump de base |
| C6 | Exploitation Docker (backup, health, séparation de conteneurs) | 2 | features ENG additionnelles |
| C7 | Support de l'écosystème Go | 2 | driver mature, tests faciles |
| C8 | File de jobs sans broker | 2 | retry + dead letter sans composant en plus |
| C9 | Courbe d'apprentissage de l'équipe | 1 | délai de 1 semaine pour le Plan |

## 3. Comparaison

### 3.1 Vue d'ensemble

| | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|
| Modèle | relationnel (+ JSONB) | relationnel embarqué | documents (BSON) |
| Architecture | serveur | fichier, dans le processus | serveur |
| Persistance | WAL, durable | journal (WAL ou rollback), durable | journal + checkpoints, durable |
| Transactions | complètes, multi-tables | complètes | multi-documents uniquement en replica set |
| Écriture concurrente | MVCC, nombreux écrivains | un seul écrivain à la fois | verrouillage au niveau document |
| Contrainte d'unicité | oui | oui | index unique |
| `INSERT … ON CONFLICT` | oui | oui (≥ 3.24) | upsert / `E11000` |
| Réclamation de jobs sans doublon | `FOR UPDATE SKIP LOCKED` | non (verrou global) | `findOneAndUpdate` atomique |
| Driver Go | `pgx` (très mature) | `mattn/go-sqlite3` (cgo) ou `modernc.org/sqlite` (pur Go) | `mongo-go-driver` (officiel) |
| Licence | PostgreSQL (permissive) | domaine public | SSPL (serveur) |
| Accès depuis plusieurs conteneurs | oui (réseau) | non pratique (fichier partagé) | oui (réseau) |

### 3.2 Notation (0 à 5, pondérée)

| Critère | Poids | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|---|
| C1 ACID | 3 | 5 | 5 | 3 |
| C2 Concurrence d'écriture | 3 | 5 | 2 | 4 |
| C3 Idempotence | 3 | 5 | 4 | 4 |
| C4 Modélisation | 2 | 4 | 3 | 5 |
| C5 Sécurité | 3 | 4 | 3 | 3 |
| C6 Exploitation Docker | 2 | 5 | 2 | 4 |
| C7 Écosystème Go | 2 | 5 | 4 | 4 |
| C8 File sans broker | 2 | 5 | 2 | 3 |
| C9 Apprentissage | 1 | 3 | 5 | 3 |
| Total pondéré (/105) | | 98 (≈ 93 %) | 69 (≈ 66 %) | 77 (≈ 73 %) |


## 4. Analyse par critère

### 4.1 Idempotence des hooks (C3)

Un hook détecte un événement (nouveau mail, nouveau commit…). Il doit l'enregistrer une seule fois, même si le polling le revoit.

Avec PostgreSQL :

```sql
CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    area_id     BIGINT      NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    external_id TEXT        NOT NULL,          -- identifiant côté service (id du mail, sha du commit…)
    payload     JSONB       NOT NULL,
    seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (area_id, external_id)
);
```

```go
// true si l'événement est nouveau, false s'il était déjà connu
func (s *Store) RecordEvent(ctx context.Context, areaID int64, extID string, payload []byte) (bool, error) {
    tag, err := s.pool.Exec(ctx, `
        INSERT INTO events (area_id, external_id, payload)
        VALUES ($1, $2, $3)
        ON CONFLICT (area_id, external_id) DO NOTHING`, areaID, extID, payload)
    if err != nil {
        return false, err
    }
    return tag.RowsAffected() == 1, nil
}
```

La déduplication est portée par la base (contrainte d'unicité), donc elle reste correcte même avec plusieurs instances de hook.

SQLite accepte la même requête (`ON CONFLICT DO NOTHING`). Son point faible est la concurrence (§4.2).

Avec MongoDB :

```go
_, err := coll.InsertOne(ctx, bson.M{"area_id": areaID, "external_id": extID, "payload": payload})
if mongo.IsDuplicateKeyError(err) {
    return false, nil // déjà vu
}
```

Il faut avoir créé l'index unique au préalable : `db.events.createIndex({area_id: 1, external_id: 1}, {unique: true})`.

### 4.2 Concurrence d'écriture (C2)

Dans AREA, l'API (clients), les hooks et les workers écrivent en même temps.

SQLite n'autorise qu'un écrivain. Le mode WAL permet aux lecteurs de continuer, mais les écritures se sérialisent :

```go
db, _ := sql.Open("sqlite", "file:area.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
db.SetMaxOpenConns(1) // sinon SQLITE_BUSY sous charge
```

Il faut donc brider le pool et faire passer toutes les écritures par une seule file. C'est viable à petite échelle, mais un seul processus peut tenir le fichier proprement, ce qui contraint l'architecture.

PostgreSQL (MVCC) gère les écritures parallèles sans configuration particulière.

MongoDB verrouille au niveau du document et encaisse aussi les écritures parallèles (7 972 ops/s à 16 goroutines, voir §6).

### 4.3 File de jobs, retry et dead letter sans broker (C8)

Le sujet impose que le retry ne nécessite pas de broker (ENG-RETRY). Avec PostgreSQL, la table elle-même sert de file :

```sql
CREATE TABLE jobs (
    id            BIGSERIAL PRIMARY KEY,
    run_id        BIGINT      NOT NULL,
    node_id       TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending',  -- pending | running | done | dead
    attempts      INT         NOT NULL DEFAULT 0,
    max_attempts  INT         NOT NULL DEFAULT 5,
    run_at        TIMESTAMPTZ NOT NULL DEFAULT now(),       -- backoff : repoussé à chaque échec
    idempotency_key TEXT      NOT NULL UNIQUE,              -- protège l'effet de bord
    last_error    TEXT
);
CREATE INDEX jobs_ready ON jobs (run_at) WHERE status = 'pending';
```

```sql
-- Réclamation atomique par plusieurs workers, sans double traitement
UPDATE jobs SET status = 'running', attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= now()
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;
```

```sql
-- Échec : backoff exponentiel, ou dead letter si les tentatives sont épuisées
UPDATE jobs SET
    status  = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    run_at  = now() + (interval '2 seconds' * power(2, attempts)),
    last_error = $2
WHERE id = $1;
```

Les jobs `dead` restent en base : inspectables et rejouables (`UPDATE jobs SET status='pending', attempts=0 …`). Le lien retry / persistance / idempotence demandé par le sujet est ici direct : la clé d'idempotence est une contrainte d'unicité dans la même transaction que l'état du job.

SQLite n'a pas de `SKIP LOCKED`. Avec un seul écrivain, la réclamation est sûre dans un seul processus, mais pas entre plusieurs.

MongoDB réclame les jobs avec `findOneAndUpdate`, qui est atomique, mais l'atomicité entre le job et l'effet de bord demande une transaction, donc un replica set.

### 4.4 Modélisation des workflows (C4)

Les nodes ont des paramètres hétérogènes. PostgreSQL combine colonnes relationnelles et JSONB :

```sql
CREATE TABLE nodes (
    id        BIGSERIAL PRIMARY KEY,
    area_id   BIGINT NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    kind      TEXT   NOT NULL,        -- 'action' | 'reaction' | 'filter' | …
    service   TEXT   NOT NULL,
    params    JSONB  NOT NULL,
    next_id   BIGINT REFERENCES nodes(id)
);
CREATE INDEX nodes_params_gin ON nodes USING GIN (params);
```

MongoDB est plus naturel pour un document workflow imbriqué, mais l'essentiel des relations d'AREA (utilisateur ↔ services ↔ workflows ↔ runs) est relationnel, avec des jointures et des suppressions en cascade (compte supprimé = données supprimées). SQLite propose du JSON (`json_extract`) mais sans indexation aussi riche.

### 4.5 Sécurité (C5)

| Aspect | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|
| Authentification / rôles | rôles, `GRANT`, row-level security | aucun (droits fichier OS) | utilisateurs, RBAC |
| Chiffrement du transport | TLS | sans objet (local) | TLS |
| Chiffrement au repos | niveau colonne (`pgcrypto`) ou volume | extension payante ou volume | Enterprise ; sinon volume |
| Requêtes paramétrées | oui (`pgx`) | oui | oui (injection NoSQL via opérateurs `$` si on passe l'entrée brute) |
| Surface d'attaque | port réseau à isoler | fichier à protéger | port réseau à isoler |

Choix de conception retenu : chiffrement applicatif des tokens OAuth (AES-256-GCM, clé hors dépôt et hors base, fournie par variable d'environnement ou secret Docker). Ainsi un dump de base ne révèle que du chiffré, quel que soit le moteur.

```go
func Encrypt(key, plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key) // clé de 32 octets
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    return gcm.Seal(nonce, nonce, plaintext, nil), nil // nonce || ciphertext
}
```

Ce choix rend le critère C5 moins discriminant entre moteurs ; la différence restante tient aux rôles (PostgreSQL permet un utilisateur applicatif sans droit de `DROP`, de lecture sur `audit`, etc.).

### 4.6 Exploitation Docker (C6)

```yaml
services:
  db:
    image: postgres:17
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U area"]
      interval: 5s
  backup:
    image: postgres:17
    # pg_dump horodaté vers un volume distinct du conteneur de la base
    command: >
      sh -c 'while true; do
        pg_dump -h db -U area area | gzip > /backups/area-$$(date +%F_%H%M).sql.gz;
        sleep 86400; done'
    volumes:
      - backups:/backups
volumes:
  pgdata:
  backups:
```

- Health endpoint : un `pool.Ping(ctx)` côté Go suffit à signaler la base arrêtée.
- Backups : `pg_dump` en conteneur dédié. Avec SQLite, il faudrait copier un fichier utilisé par l'application (`.backup` / `VACUUM INTO`) depuis un autre conteneur, donc partager le volume. MongoDB passe par `mongodump`.
- Persistance Docker : les trois moteurs exigent un volume nommé, sinon un conteneur recréé perd les données.

## 5. Choix proposé

PostgreSQL, accédé via `pgx`.

Raisons spécifiques à AREA :
1. Les hooks, workers et l'API écrivent en parallèle : MVCC sans contournement (C2).
2. L'idempotence et la file de retry reposent sur des contraintes d'unicité et `SKIP LOCKED` dans la même transaction que l'état métier, sans broker (C3, C8).
3. Le modèle est majoritairement relationnel, et le JSONB couvre la variabilité des nodes (C4).
4. Un serveur séparé s'intègre naturellement aux backups, au health check et à la décomposition en conteneurs (C6).

### Alternatives rejetées, pour ce projet

- SQLite : rejeté à cause de l'écriture mono-écrivain face à des hooks, workers et API concurrents, et du partage d'un fichier entre conteneurs. Il est durable et ACID, et reste pertinent pour les tests unitaires.
- MongoDB : rejeté car le modèle est surtout relationnel, que les transactions multi-documents exigent un replica set (complexité d'exploitation pour peu de gain), et que la licence SSPL et le chiffrement au repos natif (Enterprise) sont moins favorables.

### Conséquences acceptées

- Un conteneur serveur de plus à configurer, sécuriser et sauvegarder.
- Une file de jobs en base : adaptée à notre volume, à réévaluer si le débit explose (voir ENG-ASYNC).
- Migrations de schéma à gérer (prévoir `golang-migrate` ou `goose`).

### Conditions de révision

Revoir ce choix si l'évaluation montre que le JSONB ne suffit pas pour les workflows, ou si une charge réelle dépasse ce que la base supporte en file de jobs.

## 6. Preuve pratique (POC)

Le POC est dans `bench/persistence/` (Go et Docker Compose). Il exécute les mêmes scénarios sur les trois moteurs, derrière une interface `Store` commune.

```
nix develop                        # Go, psql, sqlite3
cd bench/persistence
docker compose up -d --wait        # postgres:17 et mongo:8
go run . bench                     # scénarios 1 à 3
./crash.sh                         # scénario 4
```

Scénarios :
1. Déduplication : 5 000 identifiants distincts soumis chacun deux fois (10 000 appels, mélangés), 16 goroutines. Attendu : exactement 5 000 lignes.
2. Écritures concurrentes : 1, 4 et 16 goroutines insèrent des runs pendant 5 s. On mesure le débit et les erreurs.
3. Réclamation de jobs : 10 000 jobs, 4 workers qui réclament puis terminent. Attendu : aucun job traité deux fois.
4. Crash : 8 goroutines écrivent en continu et n'enregistrent un identifiant qu'une fois l'écriture acquittée. On tue le moteur (`docker kill`, SIGKILL), on le redémarre, puis on vérifie que chaque écriture acquittée existe.

Variantes testées : `sqlite` (WAL, `synchronous=FULL`, pool de 1 connexion), `sqlite-pool` (même réglage, pool de 16 connexions, donc sans précaution), `mongo` (`j:true`, écriture acquittée après flush du journal) et `mongo-w1` (write concern par défaut).

### Résultats mesurés (2026-11-24, moyenne de 5 exécutions)

Les valeurs entre parenthèses donnent le minimum et le maximum observés quand l'écart est notable.

| Scénario | PostgreSQL | SQLite (pool 1) | SQLite (pool 16) | MongoDB (`j:true`) |
|---|---|---|---|---|
| Lignes après déduplication (attendu 5 000) | 5 000 | 5 000 | 5 000 | 5 000 |
| Erreurs pendant la déduplication | 0 | 0 | 14 (`SQLITE_BUSY`) | 0 |
| Durée de la déduplication | 0,67 s | 11,7 s | 12,1 s | 1,5 s |
| Débit écriture, 1 goroutine (ops/s) | 870 (71 à 1 238) | 858 (832 à 878) | 789 (452 à 878) | 477 (95 à 791) |
| Débit écriture, 4 goroutines (ops/s) | 2 241 (1 153 à 2 521) | 870 | 753 (291 à 877) | 2 400 |
| Débit écriture, 16 goroutines (ops/s) | 9 929 (9 851 à 9 964) | 866 | 709 (74 à 881) | 7 777 (7 421 à 7 983) |
| Erreurs d'écriture (16 goroutines) | 0 | 0 | 6 (`SQLITE_BUSY`) | 0 |
| Jobs traités en double (attendu 0) | 0 | 0 | 0 | 0 |
| Jobs terminés en base (attendu 10 000) | 10 000 | 10 000 | 9 998 (9 996 à 10 000) | 10 000 |
| Durée du traitement des 10 000 jobs | 18,2 s (12,9 à 23,2) | 23,9 s | 26,8 s (23,9 à 37,0) | 8,3 s |

| Crash (`kill` en pleine écriture, moyenne de 5 exécutions) | Écritures acquittées | Perdues |
|---|---|---|
| PostgreSQL | 19 404 | 0 sur les 5 exécutions |
| SQLite (`synchronous=FULL`) | 3 289 | 0 sur les 5 exécutions |
| MongoDB `j:true` | 17 005 | 0 sur les 5 exécutions |
| MongoDB `w:1` (défaut) | 85 584 | 123 en moyenne (22 à 266) |

### Lecture des résultats

- Concurrence : le débit de SQLite reste autour de 860 ops/s quel que soit le nombre de goroutines, puisqu'il n'a qu'un écrivain à la fois. PostgreSQL passe de 870 à 9 929 ops/s entre 1 et 16 goroutines (le groupement des fsync profite du parallélisme). MongoDB suit la même tendance, un peu en dessous (477 à 7 777 ops/s).
- Pool SQLite naïf : avec 16 connexions, on obtient des erreurs `SQLITE_BUSY` malgré `busy_timeout=5000`, et quelques jobs restent bloqués à l'état `running` (9 998 terminés sur 10 000 en moyenne) parce que l'écriture de fin de job a échoué. Le débit devient aussi instable (74 ops/s dans le pire cas à 16 goroutines). Il faut donc brider le pool à 1 connexion et gérer les erreurs.
- À une seule goroutine, SQLite et PostgreSQL sont au même niveau en moyenne (858 contre 870 ops/s), mais PostgreSQL est très irrégulier (71 à 1 238 ops/s) alors que SQLite varie peu (832 à 878). Cela ne remet pas en cause le choix, car AREA a plusieurs écrivains concurrents.
- Idempotence : les trois moteurs gardent 5 000 lignes grâce à la contrainte d'unicité. Ce critère ne les départage pas.
- Jobs sans doublon : aucun doublon dans les trois cas, y compris sans `SKIP LOCKED` pour SQLite (grâce à l'`UPDATE … RETURNING` atomique, tant qu'il y a une seule instance de l'application).
- Durabilité MongoDB : avec le write concern par défaut, 123 écritures acquittées en moyenne (22 à 266 selon l'exécution) ont été perdues après un kill. Il faut configurer `j:true`, ce qui réduit le débit. PostgreSQL est durable sans réglage.
- Jobs MongoDB plus rapides (8,3 s contre 18,2 s) : la réclamation est une seule opération (`findOneAndUpdate`), alors que PostgreSQL attend deux fsync (réclamation, puis fin du job). Ce point joue en faveur de MongoDB. Il pèse peu face aux autres critères, mais il faudra le reconnaître en soutenance.

### Limites du protocole

- Machine unique, 5 exécutions : les moyennes donnent les ordres de grandeur. Les débits à 1 et 4 goroutines varient beaucoup d'une exécution à l'autre (fsync du disque, activité de la machine) et ne servent pas à départager les moteurs ; ceux à 16 goroutines sont stables.
- Réseau local : les conteneurs tournent sur `localhost`, donc la latence réseau des moteurs serveurs est sous-estimée par rapport à un déploiement réel.
- `docker kill` simule un crash du processus, pas une coupure de courant : le cache disque de l'OS survit. Le test vérifie la journalisation, pas la tenue du matériel.
- Schéma minimal : une ligne par run, sans les entrées/sorties complètes de chaque node.
- Versions : PostgreSQL 17, MongoDB 8, SQLite via `modernc.org/sqlite` v1.63.0 (pur Go), Go 1.27.3, `pgx` v5, `mongo-driver` v2.
