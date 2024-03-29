package server

import (
	"github.com/gin-gonic/gin"
	"log"
	_"net/http"
	"printfulapi/src/api"
	"printfulapi/src/config"
	"strconv"
)

var ReleaseMode = "true"

func StartServer(config config.HTTP) {
	engine := initEngine(config)

	log.Printf("Listening on port %d\n", config.Port)
	err := engine.RunTLS(":"+strconv.Itoa(config.Port), config.HttpsCertFile, config.HttpsKeyFile)
	log.Fatal(err)
}

func initEngine(config config.HTTP) *gin.Engine {
	if ReleaseMode == "true" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.GET("/api", api.ApiHandler)
	r.POST("/api", api.ApiHandler)

	return r
}
