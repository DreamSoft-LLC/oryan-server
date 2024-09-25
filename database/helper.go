package database

import (
	"context"
	"fmt"
	"log"

	"github.com/DreamSoft-LLC/oryan/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func FindManyDocuments(collection string, filter primitive.M, sort primitive.D) (*mongo.Cursor, error) {
	// Define the options to sort the transactions by date
	findOptions := options.Find().SetSort(sort)
	return Database.Collection(collection).Find(context.TODO(), filter, findOptions)
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

func GetAllLoansWithAssociatesAndCustomers(filter bson.D, pageSize int, offset int) ([]bson.M, error) {
	loanCollection := Database.Collection(models.Collection.Loan)
	// Define the aggregation pipeline
	pipeline := mongo.Pipeline{
		// Match stage to filter the documents
		{
			{"$match", filter},
		},
		// Lookup Associate
		{
			{"$lookup", bson.D{
				{"from", models.Collection.Associate},
				{"localField", "associate_id"},
				{"foreignField", "_id"},
				{"as", "associate"},
			}},
		},
		{
			{"$unwind", bson.D{
				{"path", "$associate"},
				{"preserveNullAndEmptyArrays", true}, // Preserve null/empty if no associate found
			}},
		},
		// Lookup Customer
		{
			{"$lookup", bson.D{
				{"from", models.Collection.Customer},
				{"localField", "customer_id"},
				{"foreignField", "_id"},
				{"as", "customer"},
			}},
		},
		{
			{"$unwind", bson.D{
				{"path", "$customer"},
				{"preserveNullAndEmptyArrays", true}, // Preserve null/empty if no customer found
			}},
		},
		// Skip stage for pagination (offset)
		{
			{"$skip", offset},
		},
		// Limit stage for pagination (page size)
		{
			{"$limit", pageSize},
		},
	}

	// Execute the aggregation with pagination options
	cursor, err := loanCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation: %w", err)
	}
	defer func() {
		if err := cursor.Close(context.TODO()); err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

	// Store the results
	var loans []bson.M
	if err := cursor.All(context.TODO(), &loans); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	return loans, nil
}

func SumDocuments(collectionName string, filter bson.M, field string, result *primitive.Decimal128) error {
	collection := Database.Collection(collectionName)

	// Aggregation pipeline to convert the string amount to double and sum it
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.M{
			"numericAmount": bson.M{"$toDouble": "$" + field}, // Convert string field to double
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$numericAmount"}, // Sum the numeric amount
		}}},
	}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		var aggregationResult struct {
			Total interface{} `bson:"total"`
		}
		err = cursor.Decode(&aggregationResult)
		if err != nil {
			return err
		}

		// Handle different types for the sum result (int32, int64, float64)
		switch v := aggregationResult.Total.(type) {
		case int32:
			*result, err = primitive.ParseDecimal128(fmt.Sprintf("%d", v))
		case int64:
			*result, err = primitive.ParseDecimal128(fmt.Sprintf("%d", v))
		case float64:
			*result, err = primitive.ParseDecimal128(fmt.Sprintf("%f", v))
		default:
			return fmt.Errorf("unexpected type for sum result: %T", v)
		}

		if err != nil {
			return err
		}
	} else {
		// No result, set the sum to 0
		*result, _ = primitive.ParseDecimal128("0")
	}

	return nil
}

// SumAllScaleTransactions sums the total amount for all scale types ("BB", "Mini", "GB") with additional filters.
func SumAllScaleTransactions(collectionName string, filter bson.M, field string, results map[string]primitive.Decimal128) error {
	collection := Database.Collection(collectionName)

	// Aggregation pipeline to filter, convert the string amount to double, and group by scaleType to sum
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}}, // Apply the additional filters
		{{Key: "$addFields", Value: bson.M{
			"numericAmount": bson.M{"$toDouble": "$" + field}, // Convert string field to double
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$toLower": "$scale"},     // Group by scale type (BB, Mini, GB)
			"total": bson.M{"$sum": "$numericAmount"}, // Sum the numeric amount per scale
		}}},
	}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	// Loop through all the results and store them in the results map
	for cursor.Next(context.TODO()) {
		var aggregationResult struct {
			ID    string      `bson:"_id"`   // scale type (BB, Mini, GB)
			Total interface{} `bson:"total"` // sum result
		}

		err := cursor.Decode(&aggregationResult)
		if err != nil {
			return err
		}

		// Handle different types for the sum result (int32, int64, float64)
		var result primitive.Decimal128
		switch v := aggregationResult.Total.(type) {
		case int32:
			result, err = primitive.ParseDecimal128(fmt.Sprintf("%d", v))
		case int64:
			result, err = primitive.ParseDecimal128(fmt.Sprintf("%d", v))
		case float64:
			result, err = primitive.ParseDecimal128(fmt.Sprintf("%f", v))
		default:
			return fmt.Errorf("unexpected type for sum result: %T", v)
		}

		if err != nil {
			return err
		}

		// Store the result in the map with scale type as the key
		results[aggregationResult.ID] = result
	}

	// Handle the case where no results are returned, initializing all to 0 if absent
	if _, ok := results["bb"]; !ok {
		results["bb"], _ = primitive.ParseDecimal128("0")
	}
	if _, ok := results["mini"]; !ok {
		results["mini"], _ = primitive.ParseDecimal128("0")
	}
	if _, ok := results["gb"]; !ok {
		results["gb"], _ = primitive.ParseDecimal128("0")
	}

	return nil
}
