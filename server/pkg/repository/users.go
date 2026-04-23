package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string        `bson:"name" json:"name"`
	Email     string        `bson:"email" json:"email"`
	Password  string        `bson:"password" json:"password"`
	Balance   float64       `bson:"balance" json:"balance"`
	Currency  string        `bson:"currency" json:"currency"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type UserRepository struct {
	coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection("users")}
}

func (r *UserRepository) Query(ctx context.Context, query bson.M) ([]*User, error) {
	cur, err := r.coll.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()
	var users []*User
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	var u User
	if err := r.coll.FindOne(context.Background(), bson.M{"email": email}).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *User) error {
	now := time.Now()
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	u.CreatedAt = now
	u.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, u)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	var u User
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*User, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var users []*User
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id bson.ObjectID, u *User) error {
	u.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": u},
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *UserRepository) UpdateBalance(ctx context.Context, id bson.ObjectID, balance float64) error {
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"balance": balance}},
	)
	return err
}
