package models

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

var ValidateStruct = validator.New(validator.WithRequiredStructEnabled())

// Transaction struct
type Transaction struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	AssociateID primitive.ObjectID `json:"associate_id" bson:"associate_id" validate:"required"` //	Associate initiating purchase
	CustomerID  primitive.ObjectID `json:"customer_id" bson:"customer_id" validate:"required"`   //	Associate initiating purchase
	Kind        string             `json:"kind" bson:"kind" validate:"required"`                 // Kind Sell or buy
	Weight      string             `json:"weight" bson:"weight" validate:"required"`             //	Weight of the mineral
	Mineral     string             `json:"mineral" bson:"mineral" validate:"required"`           // Mineral gold or diamond
	Rate        string             `json:"rate" bson:"rate" validate:"required"`                 // Rate buying rate
	Amount      string             `json:"amount" bson:"amount" validate:"required"`             // Amount money given to seller
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Associate struct
type Associate struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email       string             `json:"email" bson:"email" validate:"required,email"`
	Password    string             `json:"password" bson:"password" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	PhoneNumber string             `json:"phone_number" bson:"phoneNumber" validate:"required"`
	Address     string             `json:"address" bson:"address" validate:"required"`
	IDNumber    string             `json:"id_number" bson:"IDNumber" validate:"required"`
	Role        string             `json:"role" bson:"role" validate:"required"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at" validate:"required"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Loan struct
type Loan struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	AssociateID primitive.ObjectID `json:"associate_id" bson:"associate_id"`
	CustomerID  primitive.ObjectID `json:"customer_id" bson:"customer_id" validate:"required"`
	Amount      string             `json:"amount" bson:"amount" validate:"required"`
	Type        string             `json:"type" bson:"type" validate:"required"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Miscellaneous struct
type Miscellaneous struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`                      // Unique identifier for each miscellaneous record
	AssociateID  primitive.ObjectID `json:"associate_id" bson:"associate_id"`   // Foreign key referencing Associate
	PurchaseType string             `json:"purchase_type" bson:"purchase_type"` // Purchase type ("Buy" or "Sell")
	Description  string             `json:"description" bson:"description"`     // Description of the purchase
	Amount       float64            `json:"amount" bson:"amount"`
	CreatedAt    time.Time          `json:"created_date" bson:"created_date"`
	UpdatedAt    time.Time          `json:"updated_date" bson:"updated_date"`
}

// Customer struct
type Customer struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`                                  // Unique identifier for each customer
	CreatedBy   primitive.ObjectID `json:"created_by" bson:"created_by"`                   // ID of Associate creating customer
	Name        string             `json:"name" bson:"name" validate:"required"`           // Name of the customer
	IDNumber    string             `json:"id_number" bson:"id_number" validate:"required"` // ID number of the customer
	Phone       string             `json:"phone" bson:"phone" validate:"required"`         // Phone number of the customer
	Email       string             `json:"email" bson:"email" validate:"required"`         // Email address of the customer
	Description string             `json:"description" bson:"description"`                 // Additional description
	CreatedAt   time.Time          `json:"created_date" bson:"created_date"`
	UpdatedAt   time.Time          `json:"updated_date" bson:"updated_date"`
}
