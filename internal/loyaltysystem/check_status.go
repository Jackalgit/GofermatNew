package loyaltysystem

import (
	"github.com/Jackalgit/GofermatNew/cmd/config"
	"github.com/Jackalgit/GofermatNew/internal/jsondecoder"
	"github.com/Jackalgit/GofermatNew/internal/models"
	"log"
	"net/http"
)

func CheckStatusOrder(orderList []models.OrderStatus) ([]models.OrderStatus, map[string]models.OrderStatus) {

	var orderListCheckStatus []models.OrderStatus
	dictOrderStatusForUpdateDB := make(map[string]models.OrderStatus)

	for _, v := range orderList {

		if v.Status != "INVALID" && v.Status != "PROCESSED" {

			URLRequest := config.Config.AccrualSystem + "/api/orders/" + v.NumOrder
			response, err := http.Get(URLRequest)
			if err != nil {
				log.Printf("[Get], %q", err)
				return nil, nil
			}
			if response.StatusCode == 204 {
				orderListCheckStatus = append(
					orderListCheckStatus,
					models.OrderStatus{NumOrder: v.NumOrder, Status: v.Status, Accrual: v.Accrual, UploadedAt: v.UploadedAt})
				continue
			}

			responsLoyaltySystem, err := jsondecoder.ResponsLoyaltySystem(response.Body)
			if err != nil {
				log.Printf("[ResponsLoyaltySystem], %q", err)
				return nil, nil
			}
			response.Body.Close()
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

	return orderListCheckStatus, dictOrderStatusForUpdateDB

}
