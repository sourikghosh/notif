package endpoints

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"notif/implementation/email"
	"notif/implementation/message"
	"notif/pkg"

	"github.com/go-playground/validator/v10"
)

// Endpoints exposes all endpoints.
type Endpoints struct {
	CreateNotif pkg.Endpoint
}

// MakeEndpoints takes service and returns Endpoints
func MakeEndpoints(svc message.Service) Endpoints {
	return Endpoints{
		CreateNotif: createNotifHandler(svc),
	}
}

// createNotifHandler to recv email from http as json send the pubAck
func createNotifHandler(svc message.Service) pkg.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var body email.Entity

		data, err := ioutil.ReadAll(request.(io.Reader))
		if err != nil {
			return nil, err
		}

		if err = json.Unmarshal(data, &body); err != nil {
			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		// validation of resquest body
		v := validator.New()
		if err := v.Struct(body); err != nil {
			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		if err := body.ToListValidation(); err != nil {
			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		// publish notif event
		pubAck, err := svc.SendEmailRequest(body)
		if err != nil {
			return nil, err
		}

		return pubAck, nil
	}
}
