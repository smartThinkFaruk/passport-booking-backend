package regional_passport_office

import "github.com/go-playground/validator/v10"

type StoreRegionalPassportOffice struct {
	Code    string `json:"code" validate:"required"`
	Name    string `json:"name" validate:"required"`
	Address string `json:"address" validate:"required"`
	Mobile  string `json:"mobile" validate:"required"`
}

func (req *StoreRegionalPassportOffice) Validate() error {
	validate := validator.New()
	return validate.Struct(req)
}
