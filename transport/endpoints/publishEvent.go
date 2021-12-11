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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Endpoints exposes all endpoints.
type Endpoints struct {
	CreateNotif pkg.Endpoint
}

// MakeEndpoints takes service and returns Endpoints
func MakeEndpoints(svc message.Service, tracer trace.Tracer) Endpoints {
	return Endpoints{
		CreateNotif: createNotifHandler(svc, tracer),
	}
}

// createNotifHandler to recv email from http as json send the pubAck
func createNotifHandler(svc message.Service, tracer trace.Tracer) pkg.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, span := tracer.Start(ctx, "create-notif-handler")
		defer span.End()

		var body email.Entity

		data, err := ioutil.ReadAll(request.(io.Reader))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, err
		}

		if err = json.Unmarshal(data, &body); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		// validation of resquest body
		v := validator.New()
		if err := v.Struct(body); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		if err := body.ToListValidation(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, pkg.NotifErr{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		// publish notif event
		pubAck, err := svc.SendEmailRequest(ctx, body)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, err
		}

		return pubAck, nil
	}
}
