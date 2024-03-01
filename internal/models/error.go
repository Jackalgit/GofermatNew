package models

import (
	"github.com/google/uuid"
)

type UniqueLoginError struct {
	Login string
}

func NewUniqueLoginError(login string) error {
	return &UniqueLoginError{Login: login}
}

func (UL *UniqueLoginError) Error() string {
	return UL.Login
}

type UniqueUserIDError struct {
	UserID uuid.UUID
}

func NewUniqueUserIDError(userID uuid.UUID) error {
	return &UniqueUserIDError{UserID: userID}
}

func (UU *UniqueUserIDError) Error() string {
	return UU.UserID.String()
}

type UserIDUniqueOrderError struct {
	UserIDnumOrder string
}

func NewUserIDUniqueOrderError(userIDnumOrder string) error {
	return &UserIDUniqueOrderError{UserIDnumOrder: userIDnumOrder}
}

func (UO *UserIDUniqueOrderError) Error() string {
	return UO.UserIDnumOrder
}

type UniqueOrderError struct {
	NumOrder string
}

func NewUniqueOrderError(numOrder string) error {
	return &UniqueOrderError{NumOrder: numOrder}
}

func (UO *UniqueOrderError) Error() string {
	return UO.NumOrder
}
