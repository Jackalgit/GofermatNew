package models

import (
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type Secret struct {
	SecretKey string `envconfig:"SECRETKEY" required:"true"`
}

type DictUserIDToken map[string]string

func NewDictUserIDToken() DictUserIDToken {
	return make(DictUserIDToken)
}

func (d DictUserIDToken) AddUserID(userID string, tokenString string) {

	d[userID] = tokenString

}
