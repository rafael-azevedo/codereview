package service

import (
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// User represents a user in the system.
type User struct {
	ID       int
	Name     string
	Email    string
	Role     string
	IsActive bool
}

// UserService interacts with the user database.
type UserService struct {
	db *sql.DB
}

// NewUserService creates a new UserService.
func NewUserService(dataSourceName string) (*UserService, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	return &UserService{db: db}, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, name, email FROM users WHERE id = $1`
	if err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetActiveUsers retrieves all active users from the database.
func (s *UserService) GetActiveUsers() ([]User, error) {
	query := `SELECT id, name, email, role, is_active FROM users WHERE is_active = true`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
