package http

import (
	"net/http"

	"notif/pkg"
	"notif/transport/endpoints"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// NewHTTPService takes all the endpoints and returns handler.
func NewHTTPService(endpoints endpoints.Endpoints, log *zap.SugaredLogger, t trace.Tracer) http.Handler {

	r := gin.New()

	r.HandleMethodNotAllowed = true
	r.Use(ginzap.Ginzap(log.Desugar(), "2006/01/02 - 15:04:05", true))
	r.Use(ginzap.RecoveryWithZap(log.Desugar(), true))

	notif := r.Group("/notif-svc/v1")
	{
		notif.POST("/create", endpointRequestEncoder(endpoints.CreateNotif, t))
	}

	return r
}

// endpointRequestEncoder encodes request and does error handling
// and send response.
func endpointRequestEncoder(endpoint pkg.Endpoint, t trace.Tracer) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var statusCode int
		ctx, span := t.Start(c, "endpoint-Req-Encoder")
		defer span.End()

		// process the request with its handler
		response, err := endpoint(ctx, c.Request.Body)
		if err != nil {
			// if statusCode is not send then return InternalServerErr
			switch e := err.(type) {
			case pkg.Error:
				statusCode = e.Status()

			default:
				statusCode = http.StatusInternalServerError
			}

			c.AbortWithStatusJSON(statusCode, gin.H{
				"error":   true,
				"message": err.Error(),
			})

			return
		}

		// if err did not occur then return Ok status
		span.SetStatus(codes.Ok, "request proccessed suceessfully")
		c.JSON(http.StatusOK, response)
	}

	return gin.HandlerFunc(fn)
}
