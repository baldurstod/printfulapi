package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"printfulapi/src/api"
	"printfulapi/src/config"
	"strconv"
	"time"
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

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Request-Id"},
		AllowAllOrigins : true,
		MaxAge: 12 * time.Hour,
	  }))

	r.POST("/api", api.ApiHandler)

	return r
}
