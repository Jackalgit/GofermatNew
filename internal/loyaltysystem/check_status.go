package loyaltysystem

import (
	"context"
	"fmt"
	"github.com/Jackalgit/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/internal/jsondecoder"
	"github.com/Jackalgit/GofermatNew/internal/models"
	"log"
	"net/http"
	"time"
)

func CheckStatusOrder(orderList []models.OrderStatus) ([]models.OrderStatus, map[string]models.OrderStatus, error) {

	var orderListCheckStatus []models.OrderStatus
	dictOrderStatusForUpdateDB := make(map[string]models.OrderStatus)

	client := http.Client{}

	for _, v := range orderList {

		if v.Status != "INVALID" && v.Status != "PROCESSED" {

			ctx, cancelFunc := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancelFunc()

			URLRequest := config.Config.AccrualSystem + "/api/orders/" + v.NumOrder

			request, err := http.NewRequestWithContext(ctx, http.MethodGet, URLRequest, nil)
			if err != nil {
				log.Printf("[Request] Не удалось создать запрос: %q", err)
				return nil, nil, fmt.Errorf("[Request] Не удалось создать запрос: %q", err)
			}
			response, err := client.Do(request)
			if err != nil {
				log.Printf("[Do] Не удалось сделать запрос: %q", err)
				return nil, nil, fmt.Errorf("[Do] Не удалось сделать запрос: %q", err)
			}

			//response, err := http.Get(URLRequest)
			//if err != nil {
			//	log.Printf("[Get], %q", err)
			//	return nil, nil, fmt.Errorf("[GetURLRequest] %q", err)
			//}
			if response.StatusCode == 204 {
				orderListCheckStatus = append(
					orderListCheckStatus,
					models.OrderStatus{NumOrder: v.NumOrder, Status: v.Status, Accrual: v.Accrual, UploadedAt: v.UploadedAt})
				response.Body.Close()
				continue
			}
			response.Body.Close()

			responsLoyaltySystem, err := jsondecoder.ResponsLoyaltySystem(response.Body)
			if err != nil {
				log.Printf("[ResponsLoyaltySystem], %q", err)
				return nil, nil, fmt.Errorf("[ResponsLoyaltySystem] %q", err)
			}
			if v.Status != responsLoyaltySystem.Status {
				dictOrderStatusForUpdateDB[v.NumOrder] = models.OrderStatus{
					Status:  responsLoyaltySystem.Status,
					Accrual: v.Accrual + responsLoyaltySystem.Accrual}
				v.Status = responsLoyaltySystem.Status
				v.Accrual += responsLoyaltySystem.Accrual
			}
		}

		orderListCheckStatus = append(
			orderListCheckStatus,
			models.OrderStatus{NumOrder: v.NumOrder, Status: v.Status, Accrual: v.Accrual, UploadedAt: v.UploadedAt})
	}

	return orderListCheckStatus, dictOrderStatusForUpdateDB, nil

}
