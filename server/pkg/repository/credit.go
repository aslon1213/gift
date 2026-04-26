package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type FlexID struct {
	Oid   bson.ObjectID `bson:"oid,omitempty" json:"oid,omitempty"`
	Str   string        `bson:"str,omitempty" json:"str,omitempty"`
	IsOID bool          `bson:"is_oid" json:"is_oid"`
}

func FlexIDFromOID(id bson.ObjectID) FlexID {
	return FlexID{Oid: id, IsOID: true}
}

func FlexIDFromString(s string) FlexID {
	return FlexID{Str: s, IsOID: false}
}

func GetFlexIDFromStr(s string) FlexID {
	iod, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return FlexID{
			IsOID: false,
			Str:   s,
		}
	}
	return FlexID{
		Oid:   iod,
		IsOID: true,
	}
}

// FinanceRequestType enumerates the four kinds of mutation that a
// counterparty can propose on a Credit. Only credits where both From and
// To are registered users (OIDs) use this workflow — single-OID credits
// are mutated directly by the registered side.
type FinanceRequestType string

const (
	FinanceRequestIncreaseAmount         FinanceRequestType = "increase_amount"
	FinanceRequestDecreaseAmount         FinanceRequestType = "decrease_amount"
	FinanceRequestIncreaseResolvedAmount FinanceRequestType = "increase_resolved_amount"
	FinanceRequestDecreaseResolvedAmount FinanceRequestType = "decrease_resolved_amount"
)

// FinanceRequestStatus tracks the lifecycle of a request: pending until the
// counterparty decides, then approved (delta applied) or rejected (no-op).
type FinanceRequestStatus string

const (
	FinanceRequestPending  FinanceRequestStatus = "pending"
	FinanceRequestApproved FinanceRequestStatus = "approved"
	FinanceRequestRejected FinanceRequestStatus = "rejected"
)

type FinanceRequest struct {
	ID          bson.ObjectID        `bson:"_id" json:"id"`
	Type        FinanceRequestType   `bson:"type" json:"type"`
	Amount      float64              `bson:"amount" json:"amount"`
	Description string               `bson:"description" json:"description"`
	RequestedBy bson.ObjectID        `bson:"requested_by" json:"requested_by"`
	Status      FinanceRequestStatus `bson:"status" json:"status"`
	DecidedBy   bson.ObjectID        `bson:"decided_by,omitempty" json:"decided_by,omitempty"`
	DecidedAt   time.Time            `bson:"decided_at,omitempty" json:"decided_at,omitempty"`
	CreatedAt   time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time            `bson:"updated_at" json:"updated_at"`
}

type Credit struct {
	ID              bson.ObjectID    `bson:"_id,omitempty" json:"id,omitempty"`
	To              FlexID           `bson:"to" json:"to"`
	From            FlexID           `bson:"from" json:"from"`
	Amount          float64          `bson:"amount" json:"amount"`
	ResolvedAmount  float64          `bson:"resolved_amount" json:"resolved_amount"`
	Currency        string           `bson:"currency" json:"currency"`
	Description     string           `bson:"description" json:"description"`
	Resolved        bool             `bson:"resolved" json:"resolved"`
	Date            time.Time        `bson:"date" json:"date"`
	FinanceRequests []FinanceRequest `bson:"finance_requests" json:"finance_requests"`
	CreatedAt       time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at" json:"updated_at"`
}

// IsTwoParty reports whether both sides are registered users — i.e. the
// FinanceRequest review workflow applies.
func (c *Credit) IsTwoParty() bool {
	return c.To.IsOID && c.From.IsOID
}

// HasParty reports whether the given user is on either side of the credit.
func (c *Credit) HasParty(userID bson.ObjectID) bool {
	if c.To.IsOID && c.To.Oid == userID {
		return true
	}
	if c.From.IsOID && c.From.Oid == userID {
		return true
	}
	return false
}

// Counterparty returns the OID of the other registered party for a two-OID
// credit. Result is meaningless if !IsTwoParty.
func (c *Credit) Counterparty(userID bson.ObjectID) bson.ObjectID {
	if c.From.Oid == userID {
		return c.To.Oid
	}
	return c.From.Oid
}

// CreditSummary aggregates a user's credit position — borrowings (money the
// user owes) versus lendings (money owed to the user). The frontend renders
// these directly.
type CreditSummary struct {
	Borrowed            float64 `json:"borrowed"`
	Lent                float64 `json:"lent"`
	OutstandingBorrowed float64 `json:"outstanding_borrowed"`
	OutstandingLent     float64 `json:"outstanding_lent"`
	NetCredit           float64 `json:"net_credit"`
}

type CreditRepository struct {
	coll *mongo.Collection
}

func NewCreditRepository(db *mongo.Database) *CreditRepository {
	return &CreditRepository{coll: db.Collection("credits")}
}

func (r *CreditRepository) Create(ctx context.Context, c *Credit) error {
	now := time.Now()
	c.ID = bson.NewObjectID()

	c.CreatedAt = now
	c.UpdatedAt = now
	_, err := r.coll.InsertOne(ctx, c)
	return err
}
func (r *CreditRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Credit, error) {
	var c Credit
	if err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CreditRepository) List(ctx context.Context) ([]*Credit, error) {
	cur, err := r.coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var credits []*Credit
	if err := cur.All(ctx, &credits); err != nil {
		return nil, err
	}
	return credits, nil
}

func (r *CreditRepository) Query(ctx context.Context, filter bson.M) ([]*Credit, error) {
	cur, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	var credits []*Credit
	if err := cur.All(ctx, &credits); err != nil {
		return nil, err
	}
	return credits, nil
}

func (r *CreditRepository) Update(ctx context.Context, id bson.ObjectID, c *Credit) error {
	c.UpdatedAt = time.Now()
	_, err := r.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": c},
	)
	return err
}

func (r *CreditRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// Summary computes the user's outstanding credit position by walking every
// credit record where the user appears as either the To (borrower) or the
// From (lender). Pure aggregation — no mutation.
func (r *CreditRepository) Summary(ctx context.Context, userID bson.ObjectID) (*CreditSummary, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"to.oid": userID, "to.is_oid": true},
			{"from.oid": userID, "from.is_oid": true},
		},
	}
	credits, err := r.Query(ctx, filter)
	if err != nil {
		return nil, err
	}
	s := &CreditSummary{}
	for _, c := range credits {
		outstanding := c.Amount - c.ResolvedAmount
		if outstanding < 0 {
			outstanding = 0
		}
		switch {
		case c.To.IsOID && c.To.Oid == userID:
			s.Borrowed += c.Amount
			s.OutstandingBorrowed += outstanding
		case c.From.IsOID && c.From.Oid == userID:
			s.Lent += c.Amount
			s.OutstandingLent += outstanding
		}
	}
	s.NetCredit = s.OutstandingLent - s.OutstandingBorrowed
	return s, nil
}
