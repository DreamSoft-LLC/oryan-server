package database

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

var Client *mongo.Client
var Database *mongo.Database

func ConnectMongoDB(uri string) *mongo.Database {
	// Set the server API version to Stable API v1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	// Send a ping to confirm a successful connection
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}

	fmt.Println("[ DATABASE ] [ SUCCESS ] You successfully connected to MongoDB!")

	// Assign the client and database to global variables
	Client = client
	Database = client.Database("oryan")

	return Database
}

func Init() {
	mongoURI := os.Getenv("DATABASE_URI")
	if mongoURI == "" {
		panic("DATABASE_URL environment variable is not set")
	}
	ConnectMongoDB(mongoURI)
}
