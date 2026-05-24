package utils

import "github.com/go-playground/validator/v10"

func GetValidator() *validator.Validate {
	validator := validator.New()
	return validator
}
