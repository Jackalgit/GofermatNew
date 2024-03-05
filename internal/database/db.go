package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Jackalgit/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	//_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"time"
)

const (
	ctxTimeout = 1 * time.Second
)

type DataBase struct {
	conn *sql.DB
}

func NewDataBase() (DataBase, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	conf, err := pgxpool.ParseConfig(config.Config.DatabaseDSN)
	if err != nil {
		log.Printf("[ParseConfig] %q", err)
		return DataBase{}, fmt.Errorf("[ParseConfig] %q", err)
	}

	db, err := sql.Open("pgx", stdlib.RegisterConnConfig(conf.ConnConfig))
	if err != nil {
		log.Printf("[Open DB] Не удалось установить соединение с базой данных: %q", err)
		return DataBase{}, fmt.Errorf("[Open DB] Не удалось установить соединение с базой данных: %q", err)
	}

	query := `CREATE TABLE IF NOT EXISTS userlogin (userID VARCHAR (255) not null unique, login VARCHAR (255) not null unique, hashed_password VARCHAR (255) not null)`

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("[Create Table] Не удалось создать таблицу loging в база данных: %q", err)
		return DataBase{}, fmt.Errorf("[Create Table] Не удалось создать таблицу loging в база данных: %q", err)
	}

	// PostgreSQL хранит timestamptz значение в формате UTC, но когда данные достаются из timestamptz столбца,
	// PostgreSQL преобразует значение UTC обратно во значение времени часового пояса,
	// установленное сервером базы данных, пользователем или текущим подключением к базе данных.

	query = `CREATE TABLE IF NOT EXISTS userinfo (userID VARCHAR (255), numOrder VARCHAR (255), status VARCHAR (255), accrual FLOAT, uploaded_at TIMESTAMPTZ)`

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("[Create Table] Не удалось создать таблицу userinfo в база данных: %q", err)
		return DataBase{}, fmt.Errorf("[Create Table] Не удалось создать таблицу userinfo в база данных: %q", err)

	}

	_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX login_idx ON userlogin (login)`)
	if err != nil {
		log.Printf("[ExecContext] Не удалось создать индекс login_idx в userlogin: %q", err)
		return DataBase{}, fmt.Errorf("[ExecContext] Не удалось создать индекс login_idx в userlogin: %q", err)

	}
	_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX numOrder_idx ON userinfo (numOrder)`)
	if err != nil {
		log.Printf("[ExecContext] Не удалось создать индекс numOrder_idx в таблице userinfo: %q", err)
		return DataBase{}, fmt.Errorf("[ExecContext] Не удалось создать индекс numOrder_idx в userinfo: %q", err)

	}

	query = `CREATE TABLE IF NOT EXISTS userwithdraw (userID VARCHAR (255), numOrder VARCHAR (255), sumPoint FLOAT, processed_at TIMESTAMPTZ)`

	_, err = db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("[Create Table] Не удалось создать таблицу userwithdraw: %q", err)
		return DataBase{}, fmt.Errorf("[Create Table] Не удалось создать таблицу userwithdraw: %q", err)

	}

	return DataBase{conn: db}, nil
}

func (d DataBase) RegisterUser(ctx context.Context, userID uuid.UUID, login string, hashPass string) error {

	query := `INSERT INTO userlogin (userID, login, hashed_password) VALUES($1, $2, $3)`

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	stmt, err := d.conn.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("[PrepareContext] %s", err)
		return nil
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, login, hashPass)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {

			UniqueLoginError := models.NewUniqueLoginError(login)
			return UniqueLoginError
		}
	}

	return nil
}

func (d DataBase) LoginUser(ctx context.Context, login string) (string, string, error) {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	row := d.conn.QueryRowContext(
		ctx,
		"SELECT userID, hashed_password FROM userlogin WHERE login = $1",
		login,
	)

	var userID string
	var hashPassInDB sql.NullString

	err := row.Scan(&userID, &hashPassInDB)
	if err != nil {
		log.Printf("[row Scan] Не удалось прeобразовать данные: %q", err)
		return "", "", fmt.Errorf("[row Scan] Не удалось прeобразовать данные: %q", err)
	}

	if !hashPassInDB.Valid {
		return "", "", fmt.Errorf("пароль не найден")
	}

	return userID, hashPassInDB.String, nil
}

func (d DataBase) LoadOrderNum(ctx context.Context, userID string, numOrder string) error {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	currentTime := time.Now().Format(time.RFC3339)

	query := `INSERT INTO userinfo (userID, numOrder, status, accrual, uploaded_at) VALUES($1, $2, $3, $4, $5)`

	stmt, err := d.conn.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("[PrepareContext] %s", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, numOrder, "NEW", 0, currentTime)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {

			userIDnumOrder, err := d.GetUserIDtoNumOrder(ctx, numOrder)
			if err != nil {
				return fmt.Errorf("[GetUserIDtoNumOrder] %q", err)
			}

			userIDUniqueOrderError := models.NewUserIDUniqueOrderError(userIDnumOrder)

			return userIDUniqueOrderError

		}
		log.Println("[ExecContext LoadOrderNum]", err)
		return fmt.Errorf("[ExecContext] %q", err)
	}

	return nil
}

func (d DataBase) GetUserIDtoNumOrder(ctx context.Context, numOrder string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	row := d.conn.QueryRowContext(
		ctx,
		"SELECT userID FROM userinfo WHERE numOrder = $1", numOrder,
	)

	var userID sql.NullString
	err := row.Scan(&userID)
	if err != nil {
		log.Printf("[Scan] %v", err)
		return "", fmt.Errorf("[Scan] Не удалось прeобразовать данные: %q", err)
	}

	if !userID.Valid {
		return "", nil
	}

	return userID.String, nil

}

func (d DataBase) GetListOrder(ctx context.Context, userID string) ([]models.OrderStatus, error) {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	var orderList []models.OrderStatus
	var orderInfo models.OrderStatus

	rows, err := d.conn.QueryContext(
		ctx,
		"SELECT numOrder, status, accrual, uploaded_at FROM userinfo WHERE userID = $1",
		userID,
	)
	if err != nil {
		log.Printf("[QueryContext] Не удалось получить данные по userID: %q", err)
		return nil, fmt.Errorf("[QueryContext] Не удалось получить данные по userID: %q", err)
	}
	defer rows.Close()

	for rows.Next() {

		err = rows.Scan(&orderInfo.NumOrder, &orderInfo.Status, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			log.Printf("[rows Scan] Не удалось собрать orderInfo: %q", err)
			return nil, fmt.Errorf("[rows Scan] Не удалось собрать orderInfo: %q", err)
		}

		orderList = append(orderList, orderInfo)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		log.Printf("[rows Err]: %q", err)
		return nil, fmt.Errorf("[rows Err]: %q", err)
	}

	return orderList, nil

}

func (d DataBase) UpdateOrderStatusInDB(ctx context.Context, dictOrderStatus map[string]models.OrderStatus) error {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	query := `UPDATE userinfo SET status = $1, accrual = $2  WHERE numOrder = $3`

	stmt, err := d.conn.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("[PrepareContext] %q", err)
		return fmt.Errorf("[PrepareContext] %q", err)
	}
	defer stmt.Close()
	for numOrder, v := range dictOrderStatus {

		_, err = stmt.ExecContext(ctx, v.Status, v.Accrual, numOrder)
		if err != nil {
			log.Printf("[ExecContext UpdateOrderStatusInDB] %q", err)
			return fmt.Errorf("[ExecContext] %q", err)
		}
	}

	return nil
}

func (d DataBase) SumAccrual(ctx context.Context, userID string) (float64, error) {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	row := d.conn.QueryRowContext(
		ctx,
		"SELECT SUM(accrual) AS sum_accurual FROM userinfo WHERE userID = $1",
		userID,
	)

	var sumAccurual sql.NullFloat64

	err := row.Scan(&sumAccurual)
	if err != nil {
		log.Printf("[row Scan] Не удалось прeобразовать данные: %q", err)
		return 0, fmt.Errorf("[Scan] sumAccurual %q", userID)
	}

	if !sumAccurual.Valid {
		SQLNullValidError := models.NewSQLNullValidError(fmt.Sprintf("Упользователя нет начислений %q", userID))
		return 0, SQLNullValidError
	}

	return sumAccurual.Float64, nil

}

func (d DataBase) SumWithdrawn(ctx context.Context, userID string) (float64, error) {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	row := d.conn.QueryRowContext(
		ctx,
		"SELECT SUM(sumPoint) AS sum_sumPoint FROM userwithdraw WHERE userID = $1", userID,
	)

	var sumSumPoint sql.NullFloat64
	err := row.Scan(&sumSumPoint)
	if err != nil {
		log.Printf("[Scan] %q", err)
		return 0, fmt.Errorf("[Scan] sumSumPointl %q", userID)
	}

	if !sumSumPoint.Valid {
		return 0, nil
	}

	return sumSumPoint.Float64, nil

}

func (d DataBase) WithdrawUser(ctx context.Context, userID string, numOrder string, sumPoint float64) error {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	query := `INSERT INTO userwithdraw (userID, numOrder, sumPoint, processed_at) VALUES($1, $2, $3, $4)`

	currentTime := time.Now().Format(time.RFC3339)

	stmt, err := d.conn.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("[PrepareContext] %s", err)
		return nil
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, userID, numOrder, sumPoint, currentTime)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {

			UniqueOrderError := models.NewUniqueOrderError(numOrder)
			return UniqueOrderError
		}
	}

	return nil
}

func (d DataBase) WithdrawalsUser(ctx context.Context, userID string) []models.Withdrawals {

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	var withdrawalsList []models.Withdrawals
	var withdrawals models.Withdrawals

	rows, err := d.conn.QueryContext(
		ctx,
		"SELECT numOrder, sumPoint, processed_at FROM userwithdraw WHERE userID = $1 ORDER BY processed_at ASC",
		userID,
	)
	if err != nil {
		log.Printf("[QueryContext] Не удалось получить данные по userID: %q", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {

		err = rows.Scan(&withdrawals.Order, &withdrawals.Sum, &withdrawals.ProcessedAt)
		if err != nil {
			log.Printf("[rows Scan] Не удалось собрать orderInfo: %q", err)
			return nil
		}

		withdrawalsList = append(withdrawalsList, withdrawals)
	}
	err = rows.Err()
	if err != nil {
		log.Printf("[rows Err]: %q", err)
		return nil
	}

	return withdrawalsList

}
