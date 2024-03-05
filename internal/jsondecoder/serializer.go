package jsondecoder

import (
	"encoding/json"
	"github.com/Jackalgit/GofermatNew/internal/models"
	"io"
)

func RequestRegisterToStruct(body io.Reader) (models.Register, error) {

	var register models.Register

	dec := json.NewDecoder(body)
	if err := dec.Decode(&register); err != nil {
		return register, err
	}

	return register, nil

}

func ResponsLoyaltySystem(body io.Reader) (models.ResponsLoyaltySystem, error) {

	var respons models.ResponsLoyaltySystem

	dec := json.NewDecoder(body)
	if err := dec.Decode(&respons); err != nil {
		return respons, err
	}
	return respons, nil
}

func RequestWithdraw(body io.Reader) (models.Withdraw, error) {

	var withdraw models.Withdraw

	dec := json.NewDecoder(body)
	if err := dec.Decode(&withdraw); err != nil {
		return withdraw, err
	}

	return withdraw, nil

}
