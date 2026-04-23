package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RefreshToken represents a refresh token in the system
type RefreshToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserId    bson.ObjectID `bson:"user_id" json:"user_id"`
	Token     string        `bson:"token" json:"token"`
	ExpiresAt time.Time     `bson:"expires_at" json:"expires_at"`
	Revoked   bool          `bson:"revoked" json:"revoked"`
}

// RefreshTokenRepository handles database operations for refresh tokens
type RefreshTokenRepository struct {
	coll *mongo.Collection
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *mongo.Database) *RefreshTokenRepository {
	return &RefreshTokenRepository{coll: db.Collection("refresh_tokens")}
}

// CreateRefreshToken creates a new refresh token for a user
func (r *RefreshTokenRepository) CreateRefreshToken(userId bson.ObjectID, ttl time.Duration) (*RefreshToken, error) {
	token := &RefreshToken{
		Token:     uuid.New().String(),
		UserId:    userId,
		ExpiresAt: time.Now().Add(ttl),
		Revoked:   false,
	}
	_, err := r.coll.InsertOne(context.Background(), token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// GetRefreshToken retrieves a refresh token by its token string
func (r *RefreshTokenRepository) GetRefreshToken(tokenString string) (*RefreshToken, error) {
	var token RefreshToken
	if err := r.coll.FindOne(context.Background(), bson.M{"token": tokenString}).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *RefreshTokenRepository) RevokeRefreshToken(tokenString string) error {
	_, err := r.coll.UpdateOne(context.Background(), bson.M{"token": tokenString}, bson.M{"$set": bson.M{"revoked": true}})
	if err != nil {
		return err
	}
	return nil
}

// RevokeAllUserTokens marks all refresh tokens for a user as revoked.
func (r *RefreshTokenRepository) RevokeAllUserTokens(userId bson.ObjectID) error {
	_, err := r.coll.UpdateMany(context.Background(), bson.M{"user_id": userId}, bson.M{"$set": bson.M{"revoked": true}})
	if err != nil {
		return err
	}
	return nil
}
