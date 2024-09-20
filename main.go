package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/DreamSoft-LLC/oryan/database"
	"github.com/DreamSoft-LLC/oryan/routers"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//server configs
	PORT := os.Getenv("PORT")

	if PORT == "" {
		PORT = "10000"
	}

	ADDRESS := fmt.Sprintf(":%s", PORT)

	//connect to database
	database.Init()

	// Ensure disconnection when the application exits
	defer func() {
		if err := database.Client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	// routes controller
	router := routers.SetupRouter()

	fmt.Println("Server Started on  " + ADDRESS)
	err = router.Run(ADDRESS)
	if err != nil {
		fmt.Printf("Server could not start: %s\n", err.Error())
	}

}
