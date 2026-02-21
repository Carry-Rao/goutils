package user

import (
	"database/sql"
	"errors"

	"github.com/Carry-Rao/goutils/database/kv"
)

type User struct {
	ID           int    `json:"id"`
	GroupID      int    `json:"group_id"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Contact      string `json:"contact"`
	EncryteToken string `json:"encryte_token"`
	Status       int    `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type Users struct {
	db        *sql.DB
	kv        *kv.Client
	EnableTPA bool
}

func NewUsers(db *sql.DB, kv *kv.Client, enableTPA bool) (*Users, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if kv == nil {
		return nil, errors.New("kv is nil")
	}
	db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		group_id INTEGER, 
		username TEXT, 
		password TEXT, 
		email TEXT UNIQUE NOT NULL, 
		phone TEXT UNIQUE, 
		contact TEXT, 
		encryte_token TEXT, 
		status INTEGER, 
		created_at INTEGER, 
		updated_at INTEGER
	)
		`)
	return &Users{
		db:        db,
		kv:        kv,
		EnableTPA: enableTPA,
	}, nil
}
