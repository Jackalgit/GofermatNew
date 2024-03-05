package util

import "unicode"

func CheckNumOrder(numOrderString string) bool {
	for _, value := range numOrderString {
		if !unicode.IsDigit(value) {
			return false
		}
	}
	return true
}
