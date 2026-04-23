package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Budget struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    bson.ObjectID `bson:"user_id" json:"user_id"`
	Category  string        `bson:"category" json:"category"`
	Amount    float64       `bson:"amount" json:"amount"`
	Currency  string        `bson:"currency" json:"currency"`
	Period    string        `bson:"period" json:"period"`
	StartDate time.Time     `bson:"start_date" json:"start_date"`
	EndDate   time.Time     `bson:"end_date" json:"end_date"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type BudgetRepository struct {
	coll *mongo.Collection
}

func NewBudgetRepository(db *mongo.Database) *BudgetRepository {
	return &BudgetRepository{coll: db.Collection("budgets")}
}

func (r *BudgetRepository) Create(ctx context.Context, b *Budget) error {
	now := time.Now()
	if b.ID.IsZero() {
		b.ID = bson.NewObjectID()
	}
	b.CreatedAt = now
	b.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, b)
	return err
}

func (r *BudgetRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Budget, error) {
	var b Budget
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BudgetRepository) List(ctx context.Context) ([]*Budget, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var budgets []*Budget
	if err := cur.All(ctx, &budgets); err != nil {
		return nil, err
	}
	return budgets, nil
}

func (r *BudgetRepository) ListByUser(ctx context.Context, userID bson.ObjectID) ([]*Budget, error) {
	cur, err := r.coll.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	budgets := make([]*Budget, 0)
	if err := cur.All(ctx, &budgets); err != nil {
		return nil, err
	}
	return budgets, nil
}

func (r *BudgetRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
	return r.coll.CountDocuments(ctx, bson.M{"user_id": userID})
}

func (r *BudgetRepository) Update(ctx context.Context, id bson.ObjectID, b *Budget) error {
	b.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": b},
	)
	return err
}

func (r *BudgetRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
