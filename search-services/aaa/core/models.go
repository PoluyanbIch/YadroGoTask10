package core

type User struct {
	ID       int64  `db:"id"`
	Login    string `db:"login"`
	PassHash string `db:"pass_hash"`
	IsAdmin  bool   `db:"is_admin"`
}

type TokenClaims struct {
	UserID  int64
	Login   string
	IsAdmin bool
	Exp     int64
}
