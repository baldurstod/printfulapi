package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	_ "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"printfulapi/src/config"
	//"printfulapi/src/model"
	"github.com/baldurstod/printful-api-model"
	"time"
)

var cancelConnect context.CancelFunc
var productsCollection *mongo.Collection

func InitPrintfulDB(config config.Database) {
	log.Println(config)
	var ctx context.Context
	ctx, cancelConnect = context.WithCancel(context.Background())
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.ConnectURI))
	if err != nil {
		log.Println(err)
		panic(err)
	}

	defer closePrintfulDB()

	productsCollection = client.Database(config.DBName).Collection("products")
}

func closePrintfulDB() {
	if cancelConnect != nil {
		cancelConnect()
	}
}

func FindProduct(productID int) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"id", productID}}

	r := productsCollection.FindOne(ctx, filter)

	product := model.Product{}
	if err := r.Decode(&product); err != nil {
		return nil, err
	}

	return &product, nil
}

func InsertProduct(product *model.Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Replace().SetUpsert(true)

	filter := bson.D{{"id", product.ID}}
	_, err := productsCollection.ReplaceOne(ctx, filter, product, opts)

	return err
}
