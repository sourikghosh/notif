package http

import (
	"net/http"

	"notif/pkg"
	"notif/transport/endpoints"

	"github.com/gin-gonic/gin"
)

// NewHTTPService takes all the endpoints and returns handler.
func NewHTTPService(endpoints endpoints.Endpoints) http.Handler {

	r := gin.New()

	r.HandleMethodNotAllowed = true
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	notif := r.Group("/notif-svc/v1")
	{
		notif.POST("/create", endpointRequestEncoder(endpoints.CreateNotif))
	}

	return r
}

// endpointRequestEncoder encodes request and does error handling
// and send response.
func endpointRequestEncoder(endpoint pkg.Endpoint) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		var statusCode int

		// process the request with its handler
		response, err := endpoint(c, c.Request.Body)
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
		c.JSON(http.StatusOK, response)
	}

	return gin.HandlerFunc(fn)
}
