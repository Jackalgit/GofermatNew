package handlers

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jackalgit/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/internal/database"
	"github.com/Jackalgit/GofermatNew/internal/jsondecoder"
	"github.com/Jackalgit/GofermatNew/internal/jwt"
	"github.com/Jackalgit/GofermatNew/internal/loyaltysystem"
	"github.com/Jackalgit/GofermatNew/internal/models"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theplant/luhn"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type GoferMat struct {
	Storage         database.DataBase
	DictUserIDToken models.DictUserIDToken
}

func (g *GoferMat) Register(w http.ResponseWriter, r *http.Request) {

	request, err := jsondecoder.RequestRegisterToStruct(r.Body)
	if err != nil {
		http.Error(w, "not read body", http.StatusBadRequest)
		return
	}

	if request.Login == "" || request.Password == "" {
		http.Error(w, "логин или пароль не передан", http.StatusBadRequest)
		return
	}

	hash := md5.Sum([]byte(request.Password))
	hashedPass := hex.EncodeToString(hash[:])
	userID := uuid.New()

	ctx := r.Context()

	if err = g.Storage.RegisterUser(ctx, userID, request.Login, hashedPass); err != nil {
		http.Error(w, "логин занят", http.StatusConflict)
		return
	}

	g.SetCookie(w, r, userID.String())

	w.WriteHeader(http.StatusOK)
}

func (g *GoferMat) Login(w http.ResponseWriter, r *http.Request) {
	request, err := jsondecoder.RequestRegisterToStruct(r.Body)
	if err != nil {
		http.Error(w, "not read body", http.StatusBadRequest)
		return
	}

	if request.Login == "" || request.Password == "" {
		http.Error(w, "логин или пароль не передан", http.StatusBadRequest)
		return
	}

	hash := md5.Sum([]byte(request.Password))
	hashedPass := hex.EncodeToString(hash[:])

	ctx := r.Context()

	userID, hashPassInDB := g.Storage.LoginUser(ctx, request.Login)

	if hashPassInDB == "" {
		http.Error(w, "логин не существует", http.StatusUnauthorized)
		return
	}
	if hashedPass != hashPassInDB {
		http.Error(w, "неверный пароль", http.StatusUnauthorized)
		return
	}

	g.SetCookie(w, r, userID)

	w.WriteHeader(http.StatusOK)
}

func (g *GoferMat) ListOrders(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	cookie, err := r.Cookie("token")

	if errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "[Orders] No Cookie", http.StatusUnauthorized)
		return
	}
	cookieStr := cookie.Value
	userID, err := jwt.GetUserID(cookieStr)

	if err != nil {
		http.Error(w, "[Orders] Token is not valid", http.StatusUnauthorized)
		return
	}
	if userID == "" {
		http.Error(w, "No User ID in token", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		orderList := g.Storage.GetListOrder(ctx, userID)

		if len(orderList) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		orderListCheckStatus, dictOrderStatusForUpdateDB, err := loyaltysystem.CheckStatusOrder(orderList)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte("Повторите запрос позже"))
			return
		}

		g.Storage.UpdateOrderStatusInDB(ctx, dictOrderStatusForUpdateDB)

		responsListJSON, err := json.Marshal(orderListCheckStatus)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responsListJSON)

		return

	}

	if r.Method == http.MethodPost {

		numOrder, err := io.ReadAll(r.Body)

		if err != nil {
			log.Println("Read numOrder ERROR: ", err)
		}
		if string(numOrder) == "" {
			http.Error(w, "номер заказа не передан в запросе", http.StatusBadRequest)
			return
		}

		numOrderInt, err := strconv.Atoi(string(numOrder))
		if err != nil {
			http.Error(w, "номер заказа не цифровой формат", http.StatusBadRequest)
			return
		}

		if !luhn.Valid(numOrderInt) {
			http.Error(w, "ошибка в номере заказа", http.StatusUnprocessableEntity)
			return
		}

		if err := g.Storage.LoadOrderNum(ctx, userID, numOrderInt); err != nil {
			var UserOrderUnique *models.UserIDUniqueOrderError
			if errors.As(err, &UserOrderUnique) {
				if err.Error() == userID {
					w.WriteHeader(http.StatusOK)
					return
				} else {
					http.Error(w, "номер заказа загружен другим пользователем", http.StatusConflict)
					return
				}
			} else {
				log.Printf("[LoadOrderNum] %q", err)
			}
		}
		w.WriteHeader(http.StatusAccepted)

		return
	}
}

func (g *GoferMat) Balance(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	cookie, err := r.Cookie("token")

	if errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "[Orders] No Cookie", http.StatusUnauthorized)
		return
	}
	cookieStr := cookie.Value
	userID, err := jwt.GetUserID(cookieStr)

	if err != nil {
		http.Error(w, "[Orders] Token is not valid", http.StatusUnauthorized)
		return
	}
	if userID == "" {
		http.Error(w, "No User ID in token", http.StatusUnauthorized)
		return
	}

	sumAccurual, err := g.Storage.SumAccrual(ctx, userID)
	log.Println(sumAccurual)

	if err != nil {
		var SQLNullValidError *models.SQLNullValidError
		if errors.As(err, &SQLNullValidError) {
			http.Error(w, err.Error(), http.StatusNoContent)
			return
		} else {
			http.Error(w, "повторите запрос позже", http.StatusNoContent)
			return
		}
	}

	sumSumPoint, err := g.Storage.SumWithdrawn(ctx, userID)
	log.Println(sumSumPoint)

	if err != nil {
		var SQLNullValidError *models.SQLNullValidError
		if errors.As(err, &SQLNullValidError) {
			http.Error(w, err.Error(), http.StatusNoContent)
			return
		} else {
			http.Error(w, "повторите запрос позже", http.StatusNoContent)
			return
		}
	}

	current := sumAccurual - sumSumPoint
	log.Println(current)

	balance := models.Balance{Current: current, Withdrawn: sumSumPoint}

	responsBalance, err := json.Marshal(balance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responsBalance)
}

func (g *GoferMat) Withdraw(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	cookie, err := r.Cookie("token")

	if errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "[Orders] No Cookie", http.StatusUnauthorized)
		return
	}
	cookieStr := cookie.Value
	userID, err := jwt.GetUserID(cookieStr)

	if err != nil {
		http.Error(w, "[Orders] Token is not valid", http.StatusUnauthorized)
		return
	}
	if userID == "" {
		http.Error(w, "No User ID in token", http.StatusUnauthorized)
		return
	}

	withdrawRequest, err := jsondecoder.RequestWithdraw(r.Body)
	if err != nil {
		http.Error(w, "not read body", http.StatusBadRequest)
		return
	}
	log.Println(withdrawRequest)

	if withdrawRequest.Order == "" || withdrawRequest.Sum == 0 {
		http.Error(w, "номер заказа или сумма к списанию не передана", http.StatusBadRequest)
		return
	}

	numOrderInt, err := strconv.Atoi(withdrawRequest.Order)
	if err != nil {
		http.Error(w, "номер заказа не цифровой формат", http.StatusBadRequest)
		return
	}

	if !luhn.Valid(numOrderInt) {
		http.Error(w, "неверный номер заказа", http.StatusUnprocessableEntity)
		return
	}

	orderList := g.Storage.GetListOrder(ctx, userID)

	_, dictOrderStatusForUpdateDB, err := loyaltysystem.CheckStatusOrder(orderList)
	if err != nil {
		http.Error(w, "повторите запрос позже", http.StatusNoContent)
		return
	}

	g.Storage.UpdateOrderStatusInDB(ctx, dictOrderStatusForUpdateDB)

	sumAccurual, err := g.Storage.SumAccrual(ctx, userID)
	if err != nil {
		http.Error(w, "повторите запрос позже", http.StatusNoContent)
		return
	}

	if withdrawRequest.Sum > sumAccurual {
		http.Error(w, "на счету недостаточно средств", http.StatusPaymentRequired)
		return

	}

	if err := g.Storage.WithdrawUser(ctx, userID, numOrderInt, withdrawRequest.Sum); err != nil {
		var UniqueOrderError *models.UniqueOrderError
		if errors.As(err, &UniqueOrderError) {
			if err.Error() == fmt.Sprint(numOrderInt) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (g *GoferMat) Withdrawals(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	cookie, err := r.Cookie("token")

	if errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "[Orders] No Cookie", http.StatusUnauthorized)
		return
	}
	cookieStr := cookie.Value
	userID, err := jwt.GetUserID(cookieStr)

	if err != nil {
		http.Error(w, "[Orders] Token is not valid", http.StatusUnauthorized)
		return
	}
	if userID == "" {
		http.Error(w, "No User ID in token", http.StatusUnauthorized)
		return
	}

	withdrawalsList := g.Storage.WithdrawalsUser(ctx, userID)

	if len(withdrawalsList) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responsWithdrawals, err := json.Marshal(withdrawalsList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responsWithdrawals)
}

func (g *GoferMat) SetCookie(w http.ResponseWriter, r *http.Request, userID string) {

	tokenString := jwt.BuildJWTString(userID)
	g.DictUserIDToken.AddUserID(userID, tokenString)

	cookie := http.Cookie{Name: "token", Value: tokenString}
	r.AddCookie(&cookie)
	http.SetCookie(w, &cookie)
}

func (g *GoferMat) PingDB(w http.ResponseWriter, r *http.Request) {

	db, err := sql.Open("pgx", config.Config.DatabaseDSN)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

}
