package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Error string `json:"error"`
}

func JSON(c *gin.Context, status int, message string) {
	c.JSON(status, Body{Error: message})
}

func BadRequest(c *gin.Context, message string) {
	JSON(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	JSON(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	JSON(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	JSON(c, http.StatusNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	JSON(c, http.StatusConflict, message)
}

func Internal(c *gin.Context) {
	JSON(c, http.StatusInternalServerError, "internal server error")
}
