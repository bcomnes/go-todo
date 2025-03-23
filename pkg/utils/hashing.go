package utils

import (
	"database/sql"
)

func HashPassword(db *sql.DB, password string) (string, error) {
	var hashed string
	err := db.QueryRow("SELECT crypt($1, gen_salt('bf'))", password).Scan(&hashed)
	if err != nil {
		return "", err
	}
	return hashed, nil
}

func CheckPassword(db *sql.DB, hash, password string) (bool, error) {
	var match bool
	err := db.QueryRow("SELECT crypt($1, $2) = $2", password, hash).Scan(&match)
	if err != nil {
		return false, err
	}
	return match, nil
}
