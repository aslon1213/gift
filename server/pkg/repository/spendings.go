package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Spending struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	GroupID     bson.ObjectID `bson:"group_id,omitempty" json:"group_id,omitempty"`
	Amount      float64       `bson:"amount" json:"amount"`
	Currency    string        `bson:"currency" json:"currency"`
	Category    string        `bson:"category" json:"category"`
	Description string        `bson:"description" json:"description"`
	Date        time.Time     `bson:"date" json:"date"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

type SpendingRepository struct {
	coll *mongo.Collection
}

func NewSpendingRepository(db *mongo.Database) *SpendingRepository {
	return &SpendingRepository{coll: db.Collection("spendings")}
}

func (r *SpendingRepository) Query(ctx context.Context, query bson.M) ([]*Spending, error) {
	cur, err := r.coll.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()
	var spendings []*Spending
	if err := cur.All(ctx, &spendings); err != nil {
		return nil, err
	}
	return spendings, nil
}

func (r *SpendingRepository) Create(ctx context.Context, s *Spending) error {
	now := time.Now()
	if s.ID.IsZero() {
		s.ID = bson.NewObjectID()
	}
	s.CreatedAt = now
	s.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, s)
	return err
}

func (r *SpendingRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Spending, error) {
	var s Spending
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SpendingRepository) List(ctx context.Context) ([]*Spending, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var spendings []*Spending
	if err := cur.All(ctx, &spendings); err != nil {
		return nil, err
	}
	return spendings, nil
}

func (r *SpendingRepository) Update(ctx context.Context, id bson.ObjectID, s *Spending) error {
	s.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": s},
	)
	return err
}

func (r *SpendingRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
