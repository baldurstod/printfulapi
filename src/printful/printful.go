package printful

import (
	"encoding/json"
	"errors"
	"github.com/baldurstod/printful-api-model"
	//"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"printfulapi/src/config"
	"printfulapi/src/mongo"
	"strconv"
	"sync"
	"time"
)

var printfulConfig config.Printful

func SetPrintfulConfig(config config.Printful) {
	printfulConfig = config
	log.Println(config)
	go initAllProducts()
}

const PRINTFUL_PRODUCTS_API = "https://api.printful.com/products"
const PRINTFUL_STORE_API = "https://api.printful.com/store"
const PRINTFUL_MOCKUP_GENERATOR_API = "https://api.printful.com/mockup-generator"
const PRINTFUL_MOCKUP_GENERATOR_API_CREATE_TASK = "https://api.printful.com/mockup-generator/create-task"
const PRINTFUL_COUNTRIES_API = "https://api.printful.com/countries"
const PRINTFUL_ORDERS_API = "https://api.printful.com/orders"
const PRINTFUL_SHIPPING_API = "https://api.printful.com/shipping"
const PRINTFUL_TAX_API = "https://api.printful.com/tax"

var mutexPerEndpoint = make(map[string]*sync.Mutex)

func addEndPoint(endPoint string) int {
	mutexPerEndpoint[endPoint] = &sync.Mutex{}
	return 0
}

var _ = addEndPoint(PRINTFUL_PRODUCTS_API)
var _ = addEndPoint(PRINTFUL_STORE_API)
var _ = addEndPoint(PRINTFUL_MOCKUP_GENERATOR_API)
var _ = addEndPoint(PRINTFUL_MOCKUP_GENERATOR_API_CREATE_TASK)
var _ = addEndPoint(PRINTFUL_COUNTRIES_API)
var _ = addEndPoint(PRINTFUL_ORDERS_API)
var _ = addEndPoint(PRINTFUL_SHIPPING_API)
var _ = addEndPoint(PRINTFUL_TAX_API)

func getRateLimited(apiURL string, path string) (*http.Response, error) {
	mutex := mutexPerEndpoint[apiURL]

	mutex.Lock()
	defer mutex.Unlock()

	u, err := url.JoinPath(apiURL, path)
	if err != nil {
		return nil, errors.New("Unable to create URL")
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	header := resp.Header
	remaining := header.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return resp, err
	}

	remain, err := strconv.Atoi(remaining)
	if err != nil {
		return nil, errors.New("Unable to get rate limit")
	}

	reset, err := strconv.Atoi(header.Get("X-Ratelimit-Reset"))
	if err != nil {
		reset = 60 // default to 60s
	}

	if remain < 1 {
		reset += 2
		time.Sleep(time.Duration(reset) * time.Second)
	}

	return resp, err
}

func initAllProducts() error {
	products, err := GetProducts()
	if err != nil {
		return err
	}

	for _, v := range products {
		log.Println("Init product", v.ID)
		_, err := GetProduct(v.ID)
		log.Println(err)

		time.Sleep(100 * time.Millisecond)
		time.Sleep(time.Hour)
	}
	return nil
}

type GetCountriesResponse struct {
	Code   int             `json:"code"`
	Result []model.Country `json:"result"`
}

func GetCountries() ([]model.Country, error) {
	resp, err := getRateLimited(PRINTFUL_COUNTRIES_API, "")
	if err != nil {
		return nil, errors.New("Unable to get printful response")
	}

	response := GetCountriesResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response")
	}

	return response.Result, nil
}

type GetProductsResponse struct {
	Code   int             `json:"code"`
	Result []model.Product `json:"result"`
}

var cachedProducts = make([]model.Product, 0)
var cachedProductsUpdated = time.Time{}

func GetProducts() ([]model.Product, error) {
	now := time.Now()
	if now.After(cachedProductsUpdated.Add(12 * time.Hour)) {
		resp, err := getRateLimited(PRINTFUL_PRODUCTS_API, "")
		if err != nil {
			return nil, errors.New("Unable to get printful response")
		}

		response := GetProductsResponse{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Println(err)
			return nil, errors.New("Unable to decode printful response")
		}

		cachedProducts = response.Result
		cachedProductsUpdated = now
	}

	return cachedProducts, nil
}

type GetProductResponse struct {
	Code   int               `json:"code"`
	Result model.ProductInfo `json:"result"`
}

func GetProduct(productID int) (*model.Product, error) {
	product, err := mongo.FindProduct(productID)
	if err == nil {
		log.Println("Getting product from base")
		return product, nil
	}

	resp, err := getRateLimited(PRINTFUL_PRODUCTS_API, "/"+strconv.Itoa(productID))
	if err != nil {
		return nil, errors.New("Unable to get printful response")
	}

	response := GetProductResponse{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response")
	}

	p := &(response.Result.Product)
	mongo.InsertProduct(p)

	return p, nil
}
