package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Group struct {
	ID        bson.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string          `bson:"name" json:"name"`
	OwnerID   bson.ObjectID   `bson:"owner_id" json:"owner_id"`
	MemberIDs []bson.ObjectID `bson:"member_ids" json:"member_ids"`
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time       `bson:"updated_at" json:"updated_at"`
}

type GroupRepository struct {
	coll *mongo.Collection
}

func NewGroupRepository(db *mongo.Database) *GroupRepository {
	return &GroupRepository{coll: db.Collection("groups")}
}

func (r *GroupRepository) Query(ctx context.Context, query bson.M) ([]*Group, error) {
	cur, err := r.coll.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()
	var groups []*Group
	if err := cur.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *GroupRepository) Create(ctx context.Context, g *Group) error {
	now := time.Now()
	if g.ID.IsZero() {
		g.ID = bson.NewObjectID()
	}
	g.CreatedAt = now
	g.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, g)
	return err
}

func (r *GroupRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Group, error) {
	var g Group
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepository) List(ctx context.Context) ([]*Group, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var groups []*Group
	if err := cur.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *GroupRepository) Update(ctx context.Context, id bson.ObjectID, g *Group) error {
	g.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": g},
	)
	return err
}

func (r *GroupRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
