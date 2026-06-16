package models

import "time"

// Role is an RBAC role. Ordered by privilege: viewer < operator < admin.
type Role string

const (
	RoleViewer   Role = "viewer"   // read-only
	RoleOperator Role = "operator" // + device writes / config / actions
	RoleAdmin    Role = "admin"    // + user management
)

// rank returns a comparable privilege level for a role.
func (r Role) rank() int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleOperator:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether this role meets or exceeds the required role.
func (r Role) AtLeast(required Role) bool { return r.rank() >= required.rank() }

// Valid reports whether the role is a known value.
func (r Role) Valid() bool { return r.rank() > 0 }

// User is a local account. PasswordHash is a bcrypt hash and is never serialised.
type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         Role      `gorm:"type:text;not null" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
