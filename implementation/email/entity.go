package email

import "errors"

var ErrEmptyToList = errors.New("toList cannot be empty")
var ErrEmptyAddr = errors.New("toList email addr cannot be empty")

type NameAddr struct {
	EmailAddr string `json:"emailAddr"`
	UserName  string `json:"userName"`
}

type Entity struct {
	FromName string     `json:"fromName" validate:"required,min=4,max=15"`
	ToList   []NameAddr `json:"toList" validate:"required"`
	Subject  string     `json:"subject"`
	Body     string     `json:"" validate:"required"`
}

func (e *Entity) ToListValidation() error {
	if len(e.ToList) < 1 {
		return ErrEmptyToList
	}

	for i := range e.ToList {
		if e.ToList[i].EmailAddr == "" {
			return ErrEmptyAddr
		}
	}

	return nil
}
