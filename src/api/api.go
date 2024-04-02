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
	case "get-products":
		err = getProducts(c)
	case "get-product":
		err = getProduct(c, request.Params)
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

func getProducts(c *gin.Context) error {
	products, err := printful.GetProducts()

	if err != nil {
		return err
	}

	jsonSuccess(c, products)

	return nil
}

func getProduct(c *gin.Context, params map[string]interface{}) error {
	product, err := printful.GetProduct(int(params["product_id"].(float64)))
	log.Println(params)

	if err != nil {
		return err
	}

	jsonSuccess(c, product)

	return nil
}
