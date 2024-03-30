package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	_ "net/http"
	"printfulapi/src/printful"
)

type ApiRequest struct {
	Action  string                 `json:"action" binding:"required"`
	Version int                    `json:"version" binding:"required"`
	Params  map[string]interface{} `json:"params"`
}

func ApiHandler(c *gin.Context) {
	var request ApiRequest
	var err error

	if err = c.ShouldBindJSON(&request); err != nil {
		log.Println(err)
		jsonError(c, errors.New("Bad request"))
		return
	}

	switch request.Action {
	case "get-countries":
		err = getCountries(c)
	default:
		jsonError(c, NotFoundError{})
		return
	}

	if err != nil {
		jsonError(c, err)
	}
}

func getCountries(c *gin.Context) error {
	countries, err := printful.GetCountries()

	if err != nil {
		return err
	}

	jsonSuccess(c, countries)

	return nil
}
