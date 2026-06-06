package models

import "time"

// User represents a system operator or cashier (Spanish: Usuario).
type User struct {
	// ID is the unique identifier (Spanish: ID).
	ID int64 `json:"id" db:"id"`
	// FullName is the operator's real name (Spanish: Nombre Completo).
	FullName string `json:"fullName" db:"full_name"`
	// Username is the login identifier (Spanish: Nombre de Usuario).
	Username string `json:"username" db:"username"`
	// PasswordHash is the secure cryptographic password hash (Spanish: Hash de Contraseña).
	PasswordHash string `json:"-" db:"password_hash"`
	// Role is the authorization role (e.g. Administrator, Cajero) (Spanish: Rol).
	Role string `json:"role" db:"role"`
	// IsActive defines if the user account is enabled (Spanish: Activo).
	IsActive bool `json:"isActive" db:"is_active"`
	// CreatedAt is the registration timestamp (Spanish: Creado En).
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}
