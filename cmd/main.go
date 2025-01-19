package main

import (
	"log"
	"os"

	_ "example.com/kaisheng"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("./.env")
	port := "8080"
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	if envport := os.Getenv("LOCAL_SERVER_PORT"); envport != "" {
		port = envport
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}

}
