// POC de persistance : mêmes scénarios sur PostgreSQL, SQLite et MongoDB.
//
//	go run . bench [store ...]              scénarios 1 à 3
//	go run . crash-writer <store> <ackfile> écrit jusqu'à être tué
//	go run . crash-verify <store> <ackfile> vérifie que chaque écriture acquittée existe
package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func open(ctx context.Context, name string) (Store, error) {
	db := filepath.Join(os.TempDir(), "bench-sqlite.db")
	switch name {
	case "postgres":
		return newPG(ctx, "postgres://area:area@localhost:55432/area")
	case "sqlite": // pool de 1 connexion : configuration réglée
		return newSQLite(db, name, 1)
	case "sqlite-pool": // pool de 16 connexions : configuration naïve
		return newSQLite(db, name, 16)
	case "mongo": // j:true
		return newMongo(ctx, "mongodb://localhost:57017", name, true)
	case "mongo-w1": // write concern par défaut
		return newMongo(ctx, "mongodb://localhost:57017", name, false)
	}
	return nil, fmt.Errorf("moteur inconnu : %s", name)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: bench | crash-writer <store> <ack> | crash-verify <store> <ack>")
		os.Exit(2)
	}
	ctx := context.Background()
	switch os.Args[1] {
	case "bench":
		stores := os.Args[2:]
		if len(stores) == 0 {
			stores = []string{"postgres", "sqlite", "sqlite-pool", "mongo"}
		}
		for _, n := range stores {
			s, err := open(ctx, n)
			if err != nil {
				fmt.Printf("## %s : connexion impossible : %v\n", n, err)
				continue
			}
			runBench(ctx, s)
			s.Close()
		}
	case "crash-writer":
		crashWriter(ctx, os.Args[2], os.Args[3])
	case "crash-verify":
		crashVerify(ctx, os.Args[2], os.Args[3])
	}
}

// parallel lance fn dans n goroutines et attend leur fin.
func parallel(n int, fn func()) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	wg.Wait()
}

// errTally compte les erreurs et garde la première.
type errTally struct {
	n     atomic.Int64
	first atomic.Value
}

func (e *errTally) add(err error) {
	if e.n.Add(1) == 1 {
		e.first.Store(err.Error())
	}
}

func (e *errTally) String() string {
	if e.n.Load() == 0 {
		return "0"
	}
	return fmt.Sprintf("%d (ex: %.60s)", e.n.Load(), e.first.Load())
}

func runBench(ctx context.Context, s Store) {
	fmt.Printf("\n## %s\n", s.Name())
	if err := s.Reset(ctx); err != nil {
		fmt.Println("reset:", err)
		return
	}
	scenarioDedup(ctx, s)
	scenarioWrites(ctx, s)
	scenarioJobs(ctx, s)
}

// 1. Déduplication : 5 000 identifiants distincts soumis chacun deux fois, 16 goroutines.
func scenarioDedup(ctx context.Context, s Store) {
	const distinct = 5000
	ids := make([]string, 0, 2*distinct)
	for i := 0; i < distinct; i++ {
		id := "ev-" + strconv.Itoa(i)
		ids = append(ids, id, id)
	}
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	var next, fresh atomic.Int64
	var errs errTally
	t0 := time.Now()
	parallel(16, func() {
		for i := int(next.Add(1)) - 1; i < len(ids); i = int(next.Add(1)) - 1 {
			if ok, err := s.RecordEvent(ctx, 1, ids[i]); err != nil {
				errs.add(err)
			} else if ok {
				fresh.Add(1)
			}
		}
	})
	rows, _ := s.CountEvents(ctx)
	fmt.Printf("dedup    : 10000 soumis -> %d lignes (attendu %d), %d « nouveau », erreurs %s, %.2fs\n",
		rows, distinct, fresh.Load(), &errs, time.Since(t0).Seconds())
}

// 2. Débit d'écriture de runs pendant 5 s pour 1, 4 et 16 goroutines.
var runSeq atomic.Int64

func scenarioWrites(ctx context.Context, s Store) {
	for _, workers := range []int{1, 4, 16} {
		var ops atomic.Int64
		var errs errTally
		deadline := time.Now().Add(5 * time.Second)
		parallel(workers, func() {
			for time.Now().Before(deadline) {
				if err := s.InsertRun(ctx, runSeq.Add(1)); err != nil {
					errs.add(err)
				} else {
					ops.Add(1)
				}
			}
		})
		fmt.Printf("écriture : %2d goroutines -> %7.0f ops/s, erreurs %s\n", workers, float64(ops.Load())/5, &errs)
	}
}

// 3. Réclamation de 10 000 jobs par 4 workers : aucun job ne doit être traité deux fois.
func scenarioJobs(ctx context.Context, s Store) {
	if err := s.EnqueueJobs(ctx, 10000); err != nil {
		fmt.Println("jobs     : enqueue:", err)
		return
	}
	var mu sync.Mutex
	seen := map[int64]int{}
	var errs errTally
	t0 := time.Now()
	parallel(4, func() {
		for fails := 0; fails < 200; {
			id, ok, err := s.ClaimJob(ctx)
			if err != nil {
				errs.add(err)
				fails++
				continue
			}
			if !ok {
				return
			}
			mu.Lock()
			seen[id]++
			mu.Unlock()
			if err := s.FinishJob(ctx, id); err != nil {
				errs.add(err)
			}
		}
	})
	dups := 0
	for _, c := range seen {
		dups += c - 1
	}
	done, _ := s.CountJobsDone(ctx)
	fmt.Printf("jobs     : %d traités, %d terminés en base, %d en double (attendu 0), erreurs %s, %.2fs\n",
		len(seen), done, dups, &errs, time.Since(t0).Seconds())
}

// 4. Crash : 8 goroutines écrivent jusqu'à ce que le moteur soit tué. Un identifiant
// n'est écrit dans le fichier d'acquittement qu'après la réponse du moteur.
func crashWriter(ctx context.Context, name, ackPath string) {
	s, err := open(ctx, name)
	if err == nil {
		err = s.Reset(ctx)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ack, _ := os.OpenFile(ackPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0o644)
	var mu sync.Mutex
	var seq atomic.Int64
	parallel(8, func() {
		for {
			id := seq.Add(1)
			if s.InsertRun(ctx, id) != nil {
				return // le moteur a été tué
			}
			mu.Lock()
			fmt.Fprintln(ack, id)
			mu.Unlock()
		}
	})
	fmt.Println("writer arrêté (moteur injoignable)")
}

func crashVerify(ctx context.Context, name, ackPath string) {
	s, err := open(ctx, name)
	var have map[int64]struct{}
	if err == nil {
		defer s.Close()
		have, err = s.RunIDs(ctx)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	f, _ := os.Open(ackPath)
	defer f.Close()
	acked, lost := 0, 0
	for sc := bufio.NewScanner(f); sc.Scan(); acked++ {
		id, _ := strconv.ParseInt(sc.Text(), 10, 64)
		if _, ok := have[id]; !ok {
			lost++
		}
	}
	fmt.Printf("crash    : %-10s %d écritures acquittées, %d perdues après le kill (attendu 0), %d lignes en base\n",
		name, acked, lost, len(have))
}
