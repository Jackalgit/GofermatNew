package jwt

import (
	"fmt"
	"github.com/Jackalgit/GofermatNew/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/GofermatNew/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"log"
)

//const TOKEN_EXP = time.Hour * 24

func BuildJWTString(userID string) string {

	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		// собственное утверждение
		UserID: userID,
	})
	// создаём строку токена
	tokenString, err := token.SignedString([]byte(config.Config.SecretKey))
	if err != nil {
		log.Printf("[SignedString] %q", err)
	}

	return tokenString
}

func GetUserID(tokenString string) (string, error) {
	// создаём экземпляр структуры с утверждениями
	claims := &models.Claims{}
	// парсим из строки токена tokenString в структуру claims
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(config.Config.SecretKey), nil
		})
	if err != nil {
		log.Printf("[ParseWithClaims] %q", err)
	}

	if !token.Valid {
		return claims.UserID, fmt.Errorf("token is not valid %v", token.Valid)
	}

	// возвращаем ID пользователя в читаемом виде
	return claims.UserID, nil
}
