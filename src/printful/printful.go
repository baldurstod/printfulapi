package printful

import (
	"encoding/json"
	"errors"
	printfulAPIModel "github.com/baldurstod/printful-api-model"
	//"io/ioutil"
	"bytes"
	"encoding/base64"
	"github.com/baldurstod/randstr"
	"golang.org/x/image/draw"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"printfulapi/src/config"
	"printfulapi/src/model"
	"printfulapi/src/mongo"
	"strconv"
	"strings"
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

func fetchRateLimited(method string, apiURL string, path string, headers map[string]string, body map[string]interface{}) (*http.Response, error) {
	mutex := mutexPerEndpoint[apiURL]

	mutex.Lock()
	defer mutex.Unlock()

	u, err := url.JoinPath(apiURL, path)
	if err != nil {
		return nil, errors.New("Unable to create URL")
	}

	var requestBody io.Reader
	if body != nil {
		out, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewBuffer(out)

	}

	req, err := http.NewRequest(method, u, requestBody)
	if err != nil {
		return nil, err
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
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
		_, err, fromPrintful := GetProduct(v.ID)
		if err != nil {
			log.Println(err)
		}
		if fromPrintful {
			// printful product API has a rate of 30/min
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}

type GetCountriesResponse struct {
	Code   int                        `json:"code"`
	Result []printfulAPIModel.Country `json:"result"`
}

func GetCountries() ([]printfulAPIModel.Country, error) {
	resp, err := fetchRateLimited("GET", PRINTFUL_COUNTRIES_API, "", nil, nil)
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
	Code   int                        `json:"code"`
	Result []printfulAPIModel.Product `json:"result"`
}

var cachedProducts = make([]printfulAPIModel.Product, 0)
var cachedProductsUpdated = time.Time{}

func GetProducts() ([]printfulAPIModel.Product, error) {
	now := time.Now()
	if now.After(cachedProductsUpdated.Add(12 * time.Hour)) {
		resp, err := fetchRateLimited("GET", PRINTFUL_PRODUCTS_API, "", nil, nil)
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
	Code   int                          `json:"code"`
	Result printfulAPIModel.ProductInfo `json:"result"`
}

func GetProduct(productID int) (*printfulAPIModel.ProductInfo, error, bool) {
	product, err := mongo.FindProduct(productID)
	if err == nil {
		return product, nil, false
	}

	resp, err := fetchRateLimited("GET", PRINTFUL_PRODUCTS_API, "/"+strconv.Itoa(productID), nil, nil)
	if err != nil {
		return nil, errors.New("Unable to get printful response"), false
	}

	//body, _ := ioutil.ReadAll(resp.Body)
	//log.Println(string(body))
	response := GetProductResponse{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response"), false
	}

	if response.Code != 200 {
		log.Println(err)
		return nil, errors.New("Printful returned an error"), false
	}

	p := &(response.Result)
	mongo.InsertProduct(p)

	return p, nil, true
}

type GetVariantResponse struct {
	Code   int                          `json:"code"`
	Result printfulAPIModel.VariantInfo `json:"result"`
}

func GetVariant(variantID int) (*printfulAPIModel.VariantInfo, error, bool) {
	variant, err := mongo.FindVariant(variantID)
	if err == nil {
		return variant, nil, false
	}

	resp, err := fetchRateLimited("GET", PRINTFUL_PRODUCTS_API, "/variant/"+strconv.Itoa(variantID), nil, nil)
	if err != nil {
		return nil, errors.New("Unable to get printful response"), false
	}

	response := GetVariantResponse{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response"), false
	}

	if response.Code != 200 {
		log.Println(err)
		return nil, errors.New("Printful returned an error"), false
	}

	v := &(response.Result)
	mongo.InsertVariant(v)

	return v, nil, true
}

type GetTemplatesResponse struct {
	Code   int                              `json:"code"`
	Result printfulAPIModel.ProductTemplate `json:"result"`
}

func GetTemplates(productID int) (*printfulAPIModel.ProductTemplate, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + printfulConfig.AccessToken,
	}

	resp, err := fetchRateLimited("GET", PRINTFUL_MOCKUP_GENERATOR_API, "/templates/"+strconv.Itoa(productID), headers, nil)
	if err != nil {
		return nil, errors.New("Unable to get printful response")
	}

	response := GetTemplatesResponse{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response")
	}

	if response.Code != 200 {
		log.Println(err)
		return nil, errors.New("Printful returned an error")
	}

	p := &(response.Result)

	return p, nil
}

type GetPrintfilesResponse struct {
	Code   int                            `json:"code"`
	Result printfulAPIModel.PrintfileInfo `json:"result"`
}

func GetPrintfiles(productID int) (*printfulAPIModel.PrintfileInfo, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + printfulConfig.AccessToken,
	}

	resp, err := fetchRateLimited("GET", PRINTFUL_MOCKUP_GENERATOR_API, "/printfiles/"+strconv.Itoa(productID), headers, nil)
	if err != nil {
		return nil, errors.New("Unable to get printful response")
	}

	response := GetPrintfilesResponse{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response")
	}

	if response.Code != 200 {
		log.Println(err)
		return nil, errors.New("Printful returned an error")
	}

	p := &(response.Result)

	return p, nil
}

func GetSimilarVariants(variantID int, placement string) ([]int, error) {
	variantInfo, err, _ := GetVariant(variantID)
	if err != nil {
		return nil, err
	}

	productInfo, err, _ := GetProduct(variantInfo.Product.ID)
	if err != nil {
		return nil, err
	}

	//log.Println("GetSimilarVariants", productInfo)
	printfileInfo, err := GetPrintfiles(variantInfo.Product.ID)
	if err != nil {
		return nil, err
	}

	variantsIDs := make([]int, 0)
	for _, v := range productInfo.Variants {
		//log.Println("GetSimilarVariants", v)
		secondVariantID := v.ID
		if (variantID == secondVariantID) || matchPrintFile(printfileInfo, variantID, secondVariantID, placement) {
			variantsIDs = append(variantsIDs, secondVariantID)

		}
	}

	return variantsIDs, nil
}

func matchPrintFile(printfileInfo *printfulAPIModel.PrintfileInfo, variantID1 int, variantID2 int, placement string) bool {
	//log.Println(printfileInfo)
	printfile1 := printfileInfo.GetPrintfile(variantID1, placement)
	printfile2 := printfileInfo.GetPrintfile(variantID2, placement)

	if (printfile1 != nil) && (printfile2 != nil) {
		return (printfile1.Width == printfile2.Width) && (printfile1.Height == printfile2.Height)
	}
	return false
}

func CreateSyncProduct(datas model.CreateSyncProductDatas) (*printfulAPIModel.PrintfileInfo, error) {
	//log.Println("CreateSyncProduct", datas)

	b64data := datas.Image[strings.IndexByte(datas.Image, ',')+1:] // Remove data:image/png;base64,

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64data))

	config, err := png.DecodeConfig(reader)
	if err != nil {
		return nil, err
	}

	if config.Width > 20000 || config.Height > 20000 {
		return nil, errors.New("Image too large")
	}

	img, err := png.Decode(base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64data)))
	if err != nil {
		return nil, err
	}

	newWidth, newHeight := 200, 200
	scaledImage := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.ApproxBiLinear.Scale(scaledImage, scaledImage.Bounds(), img, img.Bounds(), draw.Over, nil)

	filename := randstr.String(32)
	log.Println(filename)

	err = mongo.UploadImage(filename, img)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = mongo.UploadImage(filename+"_thumb", scaledImage)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	headers := map[string]string{
		"Authorization": "Bearer " + printfulConfig.AccessToken,
	}

	imageURL, err := url.JoinPath(printfulConfig.ImagesURL, "/", filename)
	if err != nil {
		return nil, errors.New("Unable to create image url")
	}

	syncVariants := []map[string]interface{}{}
	for _, v := range datas.Variants {
		syncVariant := map[string]interface{}{
			"variant_id":   v.VariantID,
			"external_id":  v.ExternalVariantID,
			"retail_price": v.RetailPrice,
			"files": []interface{}{
				map[string]interface{}{
					"url": imageURL,
				},
			},
		}
		syncVariants = append(syncVariants, syncVariant)
	}

	thumbnailURL, err := url.JoinPath(printfulConfig.ImagesURL, "/", filename+"_thumb")
	if err != nil {
		return nil, errors.New("Unable to create thumbnail url")
	}

	body := map[string]interface{}{
		"sync_product": map[string]interface{}{
			"name":      datas.Name,
			"thumbnail": thumbnailURL,
		},
		"sync_variants": syncVariants,
	}


	log.Println(body)

	resp, err := fetchRateLimited("POST", PRINTFUL_STORE_API, "/products", headers, body)
	if err != nil {
		return nil, errors.New("Unable to get printful response")
	}

	response := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Unable to decode printful response")
	}

	log.Println(response)

	return nil, nil
}
