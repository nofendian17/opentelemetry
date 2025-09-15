package postgres

import (
	"go-app/internal/domain/entity"
	"time"

	"gorm.io/gorm"
)

// UserModel represents the GORM model for users table
type UserModel struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Email     string         `gorm:"size:100;not null;uniqueIndex" json:"email"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for GORM
func (u *UserModel) TableName() string {
	return "users"
}

// ToEntity converts GORM model to domain entity
func (u *UserModel) ToEntity() (*entity.User, error) {
	user, err := entity.NewUser(u.Name, u.Email)
	if err != nil {
		return nil, err
	}
	user.SetID(entity.UserID(u.ID))
	return user, nil
}

// FromEntity converts domain entity to GORM model
func (u *UserModel) FromEntity(user *entity.User) {
	if user.ID().IsValid() {
		u.ID = uint(user.ID())
	}
	u.Name = user.Name().String()
	u.Email = user.Email().String()
}

// NewUserModelFromEntity creates a new UserModel from domain entity
func NewUserModelFromEntity(user *entity.User) *UserModel {
	model := &UserModel{}
	model.FromEntity(user)
	return model
}
