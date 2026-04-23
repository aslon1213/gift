package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Goal struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID        bson.ObjectID `bson:"user_id" json:"user_id"`
	Name          string        `bson:"name" json:"name"`
	Description   string        `bson:"description" json:"description"`
	TargetAmount  float64       `bson:"target_amount" json:"target_amount"`
	CurrentAmount float64       `bson:"current_amount" json:"current_amount"`
	Currency      string        `bson:"currency" json:"currency"`
	Deadline      time.Time     `bson:"deadline" json:"deadline"`
	CreatedAt     time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at" json:"updated_at"`
}

type GoalRepository struct {
	coll *mongo.Collection
}

func NewGoalRepository(db *mongo.Database) *GoalRepository {
	return &GoalRepository{coll: db.Collection("goals")}
}

func (r *GoalRepository) Create(ctx context.Context, g *Goal) error {
	now := time.Now()
	if g.ID.IsZero() {
		g.ID = bson.NewObjectID()
	}
	g.CreatedAt = now
	g.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, g)
	return err
}

func (r *GoalRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Goal, error) {
	var g Goal
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GoalRepository) List(ctx context.Context) ([]*Goal, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var goals []*Goal
	if err := cur.All(ctx, &goals); err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *GoalRepository) Update(ctx context.Context, id bson.ObjectID, g *Goal) error {
	g.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": g},
	)
	return err
}

func (r *GoalRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
