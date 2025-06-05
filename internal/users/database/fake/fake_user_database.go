package fake

import (
	"github.com/OliverSchlueter/mail-server/internal/users"
	"sync"
)

type DB struct {
	Items map[string]users.User
	mu    sync.Mutex
}

func NewDB() *DB {
	return &DB{
		Items: make(map[string]users.User),
		mu:    sync.Mutex{},
	}
}

func (db *DB) GetByName(name string) (*users.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	user, exists := db.Items[name]
	if !exists {
		return nil, users.ErrUserNotFound
	}
	return &user, nil
}

func (db *DB) GetByEmail(email string) (*users.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, user := range db.Items {
		if user.PrimaryEmail == email {
			return &user, nil
		}

		for _, userEmail := range user.Emails {
			if userEmail == email {
				return &user, nil
			}
		}
	}

	return nil, users.ErrUserNotFound
}

func (db *DB) DoesUserExistByEmail(email string) (bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, user := range db.Items {
		if user.PrimaryEmail == email {
			return true, nil
		}

		for _, userEmail := range user.Emails {
			if userEmail == email {
				return true, nil
			}
		}
	}

	return false, nil
}

func (db *DB) Insert(user users.User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.Items[user.Name]; exists {
		return users.ErrUserAlreadyExists
	}

	db.Items[user.Name] = user
	return nil
}
