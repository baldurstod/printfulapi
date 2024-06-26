package main

import (
	"encoding/json"
	"log"
	"os"
	"printfulapi/src/config"
	"printfulapi/src/mongo"
	"printfulapi/src/printful"
	"printfulapi/src/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	config := config.Config{}
	if content, err := os.ReadFile("config.json"); err == nil {
		if err = json.Unmarshal(content, &config); err == nil {
			printful.SetPrintfulConfig(config.Printful)
			mongo.InitPrintfulDB(config.Databases.Printful)
			mongo.InitImagesDB(config.Databases.Images)
			server.StartServer(config.HTTP)
		} else {
			log.Println("Error while reading configuration", err)
		}
	} else {
		log.Println("Error while reading configuration file", err)
	}
}
