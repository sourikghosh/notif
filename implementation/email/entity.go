package email

import "errors"

type NameAddr struct {
	EmailAddr string
	UserName  string
}

type Entity struct {
	FromName string     `validate:"required,min=4,max=15"`
	ToList   []NameAddr `validate:"required"`
	Subject  string
	Body     string `validate:"required"`
}

func (e *Entity) ToListValidation() error {
	if len(e.ToList) < 1 {
		return errors.New("toList cannot be empty")
	}

	for i := range e.ToList {
		if e.ToList[i].EmailAddr == "" {
			return errors.New("toList email addr cannot be empty")
		}
	}

	return nil
}
