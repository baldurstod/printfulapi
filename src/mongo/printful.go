package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	_ "go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"printfulapi/src/config"
	"github.com/baldurstod/printful-api-model"
	"time"
)

var cancelConnect context.CancelFunc
var productsCollection *mongo.Collection

var cacheMaxAge = 86400

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

type MongoProduct struct {
	ID          int           `json:"id" bson:"id"`
	LastUpdated int64         `json:"last_updated" bson:"last_updated"`
	Product     model.Product `json:"product"`
}

func FindProduct(productID int) (*model.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"id", productID}}

	r := productsCollection.FindOne(ctx, filter)

	doc := MongoProduct{}
	if err := r.Decode(&doc); err != nil {
		return nil, err
	}

	if time.Now().Unix() - doc.LastUpdated > cacheMaxAge {
		return &doc.Product, MaxAgeError{}
	}

	return &doc.Product, nil
}

func InsertProduct(product *model.Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Replace().SetUpsert(true)

	filter := bson.D{{"id", product.ID}}
	doc := MongoProduct{ID: product.ID, LastUpdated: time.Now().Unix(), Product: *product}
	_, err := productsCollection.ReplaceOne(ctx, filter, doc, opts)

	return err
}
