package models

type User struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Websites []Website `json:"websites"` // Slice of websites owned by the user
}

type UserInsert struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
