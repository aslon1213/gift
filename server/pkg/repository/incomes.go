package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Income struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	Amount      float64       `bson:"amount" json:"amount"`
	Currency    string        `bson:"currency" json:"currency"`
	Source      string        `bson:"source" json:"source"`
	Description string        `bson:"description" json:"description"`
	Date        time.Time     `bson:"date" json:"date"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

type IncomeRepository struct {
	coll *mongo.Collection
}

func NewIncomeRepository(db *mongo.Database) *IncomeRepository {
	return &IncomeRepository{coll: db.Collection("incomes")}
}

func (r *IncomeRepository) Create(ctx context.Context, i *Income) error {
	now := time.Now()
	if i.ID.IsZero() {
		i.ID = bson.NewObjectID()
	}
	i.CreatedAt = now
	i.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, i)
	return err
}

func (r *IncomeRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Income, error) {
	var i Income
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&i); err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *IncomeRepository) List(ctx context.Context) ([]*Income, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var incomes []*Income
	if err := cur.All(ctx, &incomes); err != nil {
		return nil, err
	}
	return incomes, nil
}

func (r *IncomeRepository) Update(ctx context.Context, id bson.ObjectID, i *Income) error {
	i.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": i},
	)
	return err
}

func (r *IncomeRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
