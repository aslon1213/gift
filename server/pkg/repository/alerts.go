package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Alert struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    bson.ObjectID `bson:"user_id" json:"user_id"`
	Type      string        `bson:"type" json:"type"`
	Title     string        `bson:"title" json:"title"`
	Message   string        `bson:"message" json:"message"`
	Read      bool          `bson:"read" json:"read"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type AlertRepository struct {
	coll *mongo.Collection
}

func NewAlertRepository(db *mongo.Database) *AlertRepository {
	return &AlertRepository{coll: db.Collection("alerts")}
}

func (r *AlertRepository) Create(ctx context.Context, a *Alert) error {
	now := time.Now()
	if a.ID.IsZero() {
		a.ID = bson.NewObjectID()
	}
	a.CreatedAt = now
	a.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, a)
	return err
}

func (r *AlertRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Alert, error) {
	var a Alert
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AlertRepository) List(ctx context.Context) ([]*Alert, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			// Optionally: log the error
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var alerts []*Alert
	if err := cur.All(ctx, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepository) Update(ctx context.Context, id bson.ObjectID, a *Alert) error {
	a.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": a},
	)
	return err
}

func (r *AlertRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
