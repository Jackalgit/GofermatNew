package main

import (
	"flag"
	"fmt"
	"github.com/Jackalgit/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/internal/database"
	"github.com/Jackalgit/GofermatNew/internal/handlers"
	"github.com/Jackalgit/GofermatNew/internal/models"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func init() {
	config.ConfigServerPort()
	config.ConfigLogger()
	config.ConfigDatabaseDSN()
	config.ConfigAccrualSystem()
	config.ConfigSecretKey()
}

func main() {

	flag.Parse()

	if err := runServer(); err != nil {
		log.Println("runServer ERROR: ", err)
	}

}

func runServer() error {

	storage, err := database.NewDataBase()
	if err != nil {
		return fmt.Errorf("ошибка базы данных: %q", err)
	}

	handler := &handlers.GoferMat{
		Storage:         storage,
		DictUserIDToken: models.NewDictUserIDToken(),
	}

	router := mux.NewRouter()

	router.HandleFunc("/ping", handler.PingDB).Methods("GET")
	router.HandleFunc("/api/user/register", handler.Register).Methods("POST")
	router.HandleFunc("/api/user/login", handler.Login).Methods("POST")
	router.HandleFunc("/api/user/orders", handler.GetListOrders).Methods("GET")
	router.HandleFunc("/api/user/orders", handler.AddOrder).Methods("POST")
	router.HandleFunc("/api/user/balance", handler.Balance).Methods("GET")
	router.HandleFunc("/api/user/balance/withdraw", handler.Withdraw).Methods("POST")
	router.HandleFunc("/api/user/withdrawals", handler.Withdrawals).Methods("GET")

	if err := http.ListenAndServe(config.Config.ServerPort, router); err != nil {
		log.Println("[ListenAndServe]:", err)
		return fmt.Errorf("[ListenAndServe]: %q", err)

	}

	return http.ListenAndServe(config.Config.ServerPort, router)

}
