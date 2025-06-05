package users

type User struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Password     string   `json:"password"`
	PrimaryEmail string   `json:"primary_email"`
	Emails       []string `json:"emails"`
}
