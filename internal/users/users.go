package users

import (
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
)

type DB interface {
	GetByName(name string) (*User, error)
	GetByEmail(email string) (*User, error)
	DoesUserExistByEmail(email string) (bool, error)
	Insert(user User) error
}

type Store struct {
	db DB
}

type Configuration struct {
	DB DB
}

func NewStore(config Configuration) *Store {
	return &Store{
		db: config.DB,
	}
}

func (s *Store) GetByName(name string) (*User, error) {
	return s.db.GetByName(name)
}

func (s *Store) GetByEmail(email string) (*User, error) {
	return s.db.GetByEmail(email)
}

func (s *Store) DoesUserExistByEmail(email string) (bool, error) {
	return s.db.DoesUserExistByEmail(email)
}

func (s *Store) Create(u User) error {
	u.ID = GenerateID()
	u.Password = Hash(u.Password)

	return s.db.Insert(u)
}

func GenerateID() string {
	return uuid.New().String()
}

func Hash(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
