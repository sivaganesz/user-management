package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash,omitempty" json:"-"`
	Name         string             `bson:"name" json:"name"`
	Role         string             `bson:"role" json:"role"`
	Team         string             `bson:"team" json:"team"`
	Permissions  []string           `bson:"permissions" json:"permissions"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
	LastLoginAt  *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
}

type UserProfile struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email       string             `bson:"email" json:"email"`
	Name        string             `bson:"name" json:"name"`
	Role        string             `bson:"role" json:"role"`
	Team        string             `bson:"team" json:"team"`
	Permissions []string           `bson:"permissions" json:"permissions"`
}

func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Role:        u.Role,
		Team:        u.Team,
		Permissions: u.Permissions,
	}
}
