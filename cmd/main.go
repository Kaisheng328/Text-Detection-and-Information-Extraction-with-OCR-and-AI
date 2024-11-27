package main

import (
	"log"
	"os"

	_ "example.com/kaisheng"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/joho/godotenv"
)

// set $env:FUNCTION_TARGET= "PostImage"
// set $env:GOOGLE_APPLICATION_CREDENTIALS= "halogen-device-438608-v9-d6edf51212ec.json"
// set $env:CGO_CFLAGS= "C:/Users/Kai/AppData/Local/Programs/Tesseract-OCR/include"
// set $env:CGO_LDFLAGS= "C:/Users/Kai/AppData/Local/Programs/Tesseract-OCR/lib"
// set $env:TESSDATA_PREFIX= "C:/Users/Kai/AppData/Local/Programs/Tesseract-OCR/tessdata"
func main() {
	err := godotenv.Load("../app.env")
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
