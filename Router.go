package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	sq "github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/api/option"
)

const openaiURL = "https://api.openai.com/v1/chat/completions"
const default_provider_name = "chatgpt"
const default_model_name = "gpt-4o-mini"
const default_prompt_message = "This is Data Retrieve from GoogleVisionOCR. PLs help me to find {type: nric | passport | driving-license | Others , number:,name:,country: {code: ,name: }, return result as **JSON (JavaScript Object Notation)** and must in stringify Json format make it machine readable message. dont use ```json. The number should not mixed with alpha.No explaination or further questions needed !!!"

type RequestBody struct {
	Base64Image string `json:"base64image"`
}
type ResponseBody struct {
	AIResponse string `json:"aiResponse"`
}

var db *sql.DB

func init() {
	functions.HTTP("PostImage", PostImage)
}

func InitSQL() error {
	var err error
	user := os.Getenv("MY_SQL_USER")
	password := os.Getenv("MY_SQL_PASSWORD")
	host := os.Getenv("MY_SQL_HOST")
	dbName := os.Getenv("MY_SQL_DB")
	tcp := os.Getenv("MY_SQL_TCP")
	log.Printf("%s:%s@%s(%s)/%s", user, password, tcp, host, dbName)
	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s", user, password, tcp, host, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error connecting to MySQL: %v", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("error pinging MySQL: %v", err)
	}

	fmt.Println("Connected to MySQL!")
	return nil
}

func PostImage(w http.ResponseWriter, r *http.Request) {
	var requestBody RequestBody
	w.Header().Set("Content-Type", "application/json")
	err := InitSQL()
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	// Parse the JSON request body
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if strings.HasPrefix(requestBody.Base64Image, "data:image/") {
		commaIndex := strings.Index(requestBody.Base64Image, ",")
		if commaIndex != -1 {
			requestBody.Base64Image = requestBody.Base64Image[commaIndex+1:]
		}
	}
	// Decode the Base64 image
	imageData, err := base64.StdEncoding.DecodeString(requestBody.Base64Image)
	if err != nil {
		http.Error(w, `{"error": "Invalid base64 image"}`, http.StatusBadRequest)
		return
	}

	ocrText, err := GetOCRText(imageData)
	if err != nil {
		log.Printf("OCR error: %v", err)
		http.Error(w, `{"error": "Failed to perform OCR"}`, http.StatusInternalServerError)
		return
	}
	formatedText := strings.ReplaceAll(ocrText, "\n", " ")
	fmt.Println("OCR Text in Single Line:", formatedText)

	imageID, err := insertImageRecord(imageData)
	if err != nil {
		log.Println("error when inserting image")
		return
	}
	// Process the extracted text with AI
	aiResponse, err := ProcessAI(formatedText)
	if err != nil {
		http.Error(w, `{"error": "Failed to process AI"}`, http.StatusInternalServerError)
		return
	}
	err = insertImageAnnotation(imageID, ocrText, formatedText, aiResponse)
	if err != nil {
		log.Println("error when inserting image annotation")
		return
	}
	log.Println("Image details Uploaded to Google Cloud and MYSQL successfully!")

	// Respond with the AI response
	responseBody := map[string]interface{}{
		"aiResponse": aiResponse,
	}
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
	}
	fmt.Println(aiResponse)
	defer db.Close()
}

func ProcessAI(formatedText string) (string, error) {
	var aiResponse string
	providerName, modelName, err := GetAIConfig() // Fetch provider and model from MySQL
	if err != nil {
		log.Printf("Error retrieving AI configuration: %v\n", err)
		return "", fmt.Errorf("error retrieving AI configuration: %v", err)
	}
	if providerName == "chatgpt" {
		aiResponse, err = ProcessChatgptAI(formatedText, modelName) // Call ChatGPT API function
		if err != nil {
			log.Printf("Error processing chatgpt AI: %v\n", err)
			return "", fmt.Errorf("error processing chatgpt AI: %v", err)
		}
	} else if providerName == "gemma" {
		aiResponse, err = ProcessGemmaAI(formatedText, modelName) // Call Gemma API function
		if err != nil {
			log.Printf("Error processing gemma AI: %v\n", err)
			return "", fmt.Errorf("error processing Gemma AI: %v", err)
		}
	} else {
		log.Printf("Unknown provider: %s\n", providerName)
		aiResponse, err = ProcessGemmaAI(formatedText, modelName)
		if err != nil {
			log.Printf("Error processing default AI: %v\n", err)
			return "", fmt.Errorf("error processing default AI: %v", err)
		}
	}
	return aiResponse, nil
}

func ProcessChatgptAI(formatedText string, modelname string) (string, error) {
	var fullResponse string
	ChatgptKey := os.Getenv("CHATGPT_KEY")
	promptMessage, err := GetPromptMessage()
	if err != nil {
		return " ", fmt.Errorf("error retrieving prompt message: %v", err)

	}
	prompt := fmt.Sprintf("%s, %s", promptMessage, formatedText)

	// Create the request body (adjust based on Ollama's API requirements)
	requestBody := map[string]interface{}{
		"model": modelname,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	// Convert the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return " ", fmt.Errorf("error marshaling JSON: %v", err)

	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", openaiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return " ", fmt.Errorf("error creating request: %v", err)
	}

	// Set the headers
	req.Header.Set("Authorization", "Bearer "+ChatgptKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return " ", fmt.Errorf("error sending request: %v", err)

	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		return " ", fmt.Errorf("response body: %v", string(body))

	}

	// Read the response stream and accumulate the content

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return " ", fmt.Errorf("error reading response body: %v", err)

	}

	// Parse the JSON response
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return " ", fmt.Errorf("error unmarshalling response body: %v", err)

	}

	// Extract and print the assistant's response content
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		if message, ok := choice["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				fullResponse = content // Assign the content directly
				return fullResponse, nil
			}
		}
	}
	return "", fmt.Errorf("no valid response from ChatGPT")
}

func ProcessGemmaAI(formatedText string, modelname string) (string, error) {
	var fullResponse string
	host := os.Getenv("OLLAMA_HOST")
	api := os.Getenv("OLLAMA_API")
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	OllamaKey := os.Getenv("OLLAMA_KEY")
	ollamaURL := fmt.Sprintf("http://%s/%s/%s", host, api, endpoint)
	promptMessage, err := GetPromptMessage()
	if err != nil {
		return " ", fmt.Errorf("error retrieving prompt message: %v", err)
	}
	prompt := fmt.Sprintf("%s, %s", promptMessage, formatedText)
	// Create the request body (adjust based on Ollama's API requirements)
	requestBody := map[string]interface{}{
		"model": modelname,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	// Convert the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return " ", fmt.Errorf("error marshaling JSON: %v", err)

	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", ollamaURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return " ", fmt.Errorf("error creating request: %v", err)

	}

	// Set the headers
	req.Header.Set("Authorization", "Bearer "+OllamaKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return " ", fmt.Errorf("error sending request: %v", err)

	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		return " ", fmt.Errorf("response body: %v", string(body))

	}

	// Read the response stream and accumulate the content

	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err == io.EOF {
			break // End of stream
		} else if err != nil {
			return " ", fmt.Errorf("error decoding response: %v", err)
		}

		if message, ok := chunk["message"]; ok {
			if content, ok := message.(map[string]interface{})["content"]; ok {
				fullResponse += content.(string)
			}
		}

		if done, ok := chunk["done"]; ok && done.(bool) {
			break
		}
	}
	return fullResponse, nil
}

func GetOCRText(imageData []byte) (string, error) {
	ctx := context.Background()
	// Create a new Vision API client
	googleCred := option.WithCredentialsFile(os.Getenv("GOOGLE_CRED"))
	client, err := vision.NewImageAnnotatorClient(ctx, googleCred)
	if err != nil {
		return "", fmt.Errorf("vision.NewImageAnnotatorClient: %v", err)
	}
	defer client.Close()

	// Create an image object
	image, err := vision.NewImageFromReader(bytes.NewReader(imageData))

	if err != nil {
		return "", fmt.Errorf("vision.NewImageFromReader: %v", err)
	}

	// Perform OCR (text detection)
	annotations, err := client.DetectTexts(ctx, image, nil, 1)
	if err != nil {
		return "", fmt.Errorf("DetectTexts: %v", err)
	}

	// Check if text was detected
	if len(annotations) == 0 {
		return "", fmt.Errorf("no text found in image")
	}

	// Return the detected text
	return annotations[0].Description, nil
}

func GetAIConfig() (string, string, error) {
	var providerName, modelName string
	// Query the AI configuration
	err := db.QueryRow("SELECT name, content FROM system_config WHERE name = 'chatgpt'").Scan(&providerName, &modelName)
	if err != nil {
		log.Println("Setting to Default Provider and Model")
		return default_provider_name, default_model_name, nil
	}

	log.Printf("Retrieved AI config: provider=%s, model=%s\n", providerName, modelName)
	return providerName, modelName, nil
}

func GetPromptMessage() (string, error) {
	var promptMessage string
	// Query the AI configuration
	err := db.QueryRow("SELECT ocr_prompt_message FROM ocr_prompt_message;").Scan(&promptMessage)
	if err != nil {
		log.Println("Attempting Default Prompt Message...")
		return default_prompt_message, nil
	}

	return promptMessage, nil
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	digits := make([]byte, length)
	for i := range digits {
		digits[i] = '0' + byte(rand.Intn(10)) // Generates a digit between '0' and '9'
	}
	return string(digits)
}

func uploadToGCS(bucketName, objectName string, imageData []byte) (string, error) {
	ctx := context.Background()

	// Create a client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return "", err
	}
	defer client.Close()

	// Get the bucket and object writer
	bucket := client.Bucket(bucketName)
	object := bucket.Object(objectName)
	writer := object.NewWriter(ctx)

	// Write image data to the object
	if _, err := io.Copy(writer, bytes.NewReader(imageData)); err != nil {
		log.Printf("Failed to write to GCS: %v", err)
		return "", err
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		log.Printf("Failed to close writer: %v", err)
		return "", err
	}

	// Make the object public (optional)
	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		log.Printf("Failed to set object ACL: %v", err)
	}

	// Return the public URL
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	return publicURL, nil
}
func insertImageAnnotation(imageID string, responseDataJSON string, responseDataRaw string, aiResponse string) error {
	location, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		log.Println("Failed to load timezone")
	}
	is_success := 0
	if responseDataJSON != "" || responseDataRaw != "" {
		is_success = 1
	}

	currentTime := time.Now().In(location)
	timestamp := currentTime.Format("2006-01-02 15:04:05")
	queryBuilder := sq.Insert("image_annotation").SetMap(map[string]interface{}{

		"image_id":           imageID,
		"type":               "text",
		"provider":           "gcp",
		"response_data_json": []byte(responseDataJSON),
		"response_data_raw":  []byte(responseDataRaw),
		"result":             []byte(aiResponse),
		"is_success":         is_success,
		"created_by":         1,
		"created_at":         timestamp,
		"updated_by":         1,
		"updated_at":         timestamp,
		"is_active":          1,
	})
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("failed to build SQL query: %v", err)
	}

	// Execute the query
	_, err = db.Exec(query, args...)
	if err != nil {
		log.Printf("failed to execute insert: %v", err)
	}
	log.Println("Record inserted successfully")
	return nil
}
func insertImageRecord(imageData []byte) (string, error) {
	var Id int
	location, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		log.Println("Failed to load timezone")
	}
	currentTime := time.Now().In(location)
	timestamp := currentTime.Format("2006-01-02 15:04:05")
	imageTime := currentTime.Unix()
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.Fatalf("Failed to decode image: %v", err)
	}
	if img == nil {
		log.Fatal("Image decode returned nil")
	}
	size := len(imageData)
	imageID := fmt.Sprintf("%d.%s", imageTime, format)
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	bucketname := os.Getenv("BUCKET_NAME")

	queryBuilder := sq.Insert("image").SetMap(map[string]interface{}{

		"name":       imageID,
		"format":     "image/" + format,
		"extension":  format,
		"size":       size,
		"width":      width,
		"height":     height,
		"position":   999,
		"is_private": false,
		"created_by": 1,
		"created_at": timestamp,
		"updated_by": 1,
		"updated_at": timestamp,
		"is_active":  1,
	})
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("failed to build SQL query: %v", err)
	}

	// Execute the query
	_, err = db.Exec(query, args...)
	if err != nil {
		log.Printf("failed to execute insert: %v", err)
	}
	log.Println("Record inserted successfully")
	err = db.QueryRow("SELECT id FROM image WHERE name = ?", imageID).Scan(&Id)
	if err != nil {
		log.Println("error retrieving id number")
	}
	randomNumbers := generateRandomString(6)
	objectname := "image/original/" + fmt.Sprintf("%v_%v_%v", Id, randomNumbers, imageID)
	src, err := uploadToGCS(bucketname, objectname, imageData)
	if err != nil {
		log.Printf("failed to upload: %v", err)
	}
	result, err := db.Exec("UPDATE image SET src = ? WHERE id = ?", src, Id)
	if err != nil {
		log.Printf("failed to update src: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Failed to get affected rows: %v", err)
	}
	fmt.Printf("Number of rows updated: %d", rowsAffected)
	return imageID, nil
}
