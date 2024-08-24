package database

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

func InsertDocument(collection string, data primitive.D) (*mongo.InsertOneResult, error) {
	return Database.Collection(collection).InsertOne(context.TODO(), data)
}

func InsertManyDocument(collection string, data []interface{}) (*mongo.InsertManyResult, error) {
	return Database.Collection(collection).InsertMany(context.Background(), data)
}

func FindDocument(collection string, filter primitive.D) *mongo.SingleResult {
	return Database.Collection(collection).FindOne(context.TODO(), filter)
}
func FindDocumentById(collection string, id string) *mongo.SingleResult {
	objectId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		log.Printf("Invalid id: %v", err)
		return nil
	}

	return Database.Collection(collection).FindOne(context.TODO(), bson.D{{
		Key: "_id", Value: objectId,
	}})

}
func FindDocuments(collection string, filter primitive.D) (*mongo.Cursor, error) {
	return Database.Collection(collection).Find(context.TODO(), filter)
}

func FindDocumentsQuery(collection string, filter primitive.D, pageSize int, offset int) (*mongo.Cursor, error) {
	// Assuming you have a `Database` variable that holds your MongoDB database connection
	coll := Database.Collection(collection)

	// Create find options for pagination
	findOptions := options.Find()
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSkip(int64(offset))

	// Execute the query
	cursor, err := coll.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func DeleteDocument(collection string, id string) *mongo.SingleResult {
	return Database.Collection(collection).FindOneAndDelete(context.TODO(), bson.D{{
		Key: "_id", Value: id,
	}})
}
func DeleteDocuments(collection string, filter primitive.D) (*mongo.DeleteResult, error) {
	return Database.Collection(collection).DeleteMany(context.TODO(), filter)
}

func UpdateDocument(collection string, filter primitive.D, update interface{}) (*mongo.UpdateResult, error) {
	return Database.Collection(collection).UpdateOne(context.TODO(), filter, update)
}

func UpdateDocuments(collection string, filter primitive.D, update interface{}) (*mongo.UpdateResult, error) {
	return Database.Collection(collection).UpdateMany(context.TODO(), filter, update)
}
