package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type mongoStore struct {
	c    *mongo.Client
	db   *mongo.Database
	name string
}

// journaled=true: writes are acknowledged after the journal is flushed (j:true).
// journaled=false: default write concern (w:1).
func newMongo(ctx context.Context, uri, name string, journaled bool) (Store, error) {
	wc := writeconcern.W1()
	if journaled {
		wc = writeconcern.Journaled()
	}
	c, err := mongo.Connect(options.Client().ApplyURI(uri).SetWriteConcern(wc).SetMaxPoolSize(20))
	if err != nil {
		return nil, err
	}
	if err := c.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return &mongoStore{c, c.Database("area"), name}, nil
}

func (s *mongoStore) Name() string                   { return s.name }
func (s *mongoStore) Close()                         { _ = s.c.Disconnect(context.Background()) }
func (s *mongoStore) col(n string) *mongo.Collection { return s.db.Collection(n) }

func (s *mongoStore) Reset(ctx context.Context) error {
	for _, n := range []string{"events", "runs", "jobs"} {
		if err := s.col(n).Drop(ctx); err != nil {
			return err
		}
	}
	_, err := s.col("events").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "area_id", Value: 1}, {Key: "external_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	_, err = s.col("jobs").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}, {Key: "_id", Value: 1}},
	})
	return err
}

func (s *mongoStore) RecordEvent(ctx context.Context, areaID int64, extID string) (bool, error) {
	_, err := s.col("events").InsertOne(ctx,
		bson.M{"area_id": areaID, "external_id": extID, "payload": bson.M{"from": "bench"}})
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *mongoStore) CountEvents(ctx context.Context) (int64, error) {
	return s.col("events").CountDocuments(ctx, bson.M{})
}

func (s *mongoStore) InsertRun(ctx context.Context, id int64) error {
	nodes := bson.A{bson.M{"n": "a", "ok": true}, bson.M{"n": "b", "ok": true}}
	_, err := s.col("runs").InsertOne(ctx,
		bson.M{"_id": id, "area_id": 1, "payload": bson.M{"run": id, "nodes": nodes}})
	return err
}

func (s *mongoStore) RunIDs(ctx context.Context) (map[int64]struct{}, error) {
	cur, err := s.col("runs").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := map[int64]struct{}{}
	for cur.Next(ctx) {
		var d struct {
			ID int64 `bson:"_id"`
		}
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		out[d.ID] = struct{}{}
	}
	return out, cur.Err()
}

func (s *mongoStore) EnqueueJobs(ctx context.Context, n int) error {
	docs := make([]any, 0, 1000)
	for i := 1; i <= n; i++ {
		docs = append(docs, bson.M{"_id": int64(i), "status": "pending", "attempts": 0})
		if len(docs) == 1000 || i == n {
			if _, err := s.col("jobs").InsertMany(ctx, docs); err != nil {
				return err
			}
			docs = docs[:0]
		}
	}
	return nil
}

func (s *mongoStore) ClaimJob(ctx context.Context) (int64, bool, error) {
	var d struct {
		ID int64 `bson:"_id"`
	}
	err := s.col("jobs").FindOneAndUpdate(ctx, bson.M{"status": "pending"},
		bson.M{"$set": bson.M{"status": "running"}, "$inc": bson.M{"attempts": 1}},
		options.FindOneAndUpdate().SetSort(bson.M{"_id": 1})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, false, nil
	}
	return d.ID, err == nil, err
}

func (s *mongoStore) FinishJob(ctx context.Context, id int64) error {
	_, err := s.col("jobs").UpdateByID(ctx, id, bson.M{"$set": bson.M{"status": "done"}})
	return err
}

func (s *mongoStore) CountJobsDone(ctx context.Context) (int64, error) {
	return s.col("jobs").CountDocuments(ctx, bson.M{"status": "done"})
}
