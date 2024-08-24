package utils

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
)

func ConvertStructPrimitive(data interface{}) primitive.D {
	// Convert the struct to a BSON document
	bsonData, err := bson.Marshal(data)
	if err != nil {
		log.Fatal("Error marshalling to BSON:", err)
	}

	// Unmarshal BSON data into primitive.D
	var primitiveData bson.D

	err = bson.Unmarshal(bsonData, &primitiveData)
	if err != nil {
		log.Fatal("Error unmarshalling BSON data:", err)
	}

	return primitiveData

}
