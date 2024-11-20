package router

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/api/option"
)

const (
	openaiURL              = "https://api.openai.com/v1/chat/completions"
	spaceURL               = "https://api.ocr.space/parse/image"
	default_prompt_message = "This is Data Retrieve from GoogleVisionOCR. PLs help me to find {type: nric | passport | driving-license | Others , number:,name:,country: {code: ,name: }, return result as **JSON (JavaScript Object Notation)** and must in stringify Json format make it machine readable message. dont use ```json. The number should not mixed with alpha.No explaination or further questions needed !!!"
)

type OCRInput struct {
	Base64Image         string `json:"base64image"`
	OCRProvider         string `json:"ocr_provider"`
	AIProviderName      string `json:"ai_provider_name"`
	AIProviderModelName string `json:"ai_provider_model_name"`
}
type ResponseBody struct {
	AIResponse string `json:"aiResponse"`
}

func init() {
	functions.HTTP("PostImage", PostImage)
}

func PostImage(w http.ResponseWriter, r *http.Request) {
	var requestBody OCRInput
	availableProviders := map[string][]string{
		"chatgpt": {"gpt-4o-mini", "gpt-3.5-turbo"},
		"gemma":   {"gemma2", "gemma1"},
	}
	w.Header().Set("Content-Type", "application/json")
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

	formatedText, err := OCRVersion(imageData, requestBody.OCRProvider)
	if err != nil {
		log.Printf("OCR error: %v", err)
		http.Error(w, `{"error": "Failed to perform OCR"}`, http.StatusInternalServerError)
		return
	}
	// formatedText := strings.ReplaceAll(ocrText, "\n", " ")
	fmt.Println("OCR Text in Single Line:", formatedText)

	if requestBody.AIProviderName == "" && requestBody.AIProviderModelName == "" {
		responseBody := map[string]interface{}{
			"ocrText": formatedText,
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}
	models, providerExists := availableProviders[requestBody.AIProviderName]
	if !providerExists {
		log.Printf("Invalid AI provider: %s", requestBody.AIProviderName)
		responseBody := map[string]interface{}{
			"ocrText": formatedText,
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}
	if requestBody.AIProviderModelName == "" {
		requestBody.AIProviderModelName = models[0]
	}
	// Process the extracted text with AI
	aiResponse, err := ProcessAI(formatedText, requestBody.AIProviderName, requestBody.AIProviderModelName)
	if err != nil {
		responseBody := map[string]interface{}{
			"ocrText": formatedText,
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}

	// Respond with the AI response
	responseBody := map[string]interface{}{
		"aiResponse": aiResponse,
	}
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
	}
	fmt.Println(aiResponse)
}

func ProcessAI(formatedText string, providerName string, modelName string) (string, error) {
	var aiResponse string
	var err error
	switch providerName {
	case "chatgpt":
		aiResponse, err = ProcessChatgptAI(formatedText, modelName) // Call ChatGPT API function
		if err != nil {
			log.Printf("Error processing chatgpt AI: %v\n", err)
			return "", fmt.Errorf("error processing chatgpt AI: %v", err)
		}
	case "gemma":
		aiResponse, err = ProcessGemmaAI(formatedText, modelName) // Call Gemma API function
		if err != nil {
			log.Printf("Error processing gemma AI: %v\n", err)
			return "", fmt.Errorf("error processing Gemma AI: %v", err)
		}
	default:
		log.Printf("Unknown provider: %s\n", providerName)
		aiResponse, err = ProcessGemmaAI(formatedText, modelName) // Default to Gemma
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
	promptMessage := default_prompt_message
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
	promptMessage := default_prompt_message
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

func OCRVersion(imageData []byte, provider string) (string, error) {
	switch provider {
	case "Google":
		result, err := GoogleOCRText(imageData)
		if err != nil {
			log.Printf("Error in GoogleOCRText: %v", err)
			return "", err
		}
		return result, nil
	case "Space":
		result, err := SpaceOCRText(imageData)
		if err != nil {
			log.Printf("Error in GoogleOCRText: %v", err)
			return "", err
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported OCR provider: %s", provider)
	}
}

func SpaceOCRText(imageData []byte) (string, error) {
	type ocrResponse struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		IsErroredOnProcessing bool     `json:"IsErroredOnProcessing"`
		ErrorMessage          []string `json:"ErrorMessage"`
	}
	apiKey := os.Getenv("SPACE_CRED")
	if apiKey == "" {
		return "", fmt.Errorf("SPACE_CRED environment variable not set")
	}

	// Create multipart form data
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Write API key and image data
	if err := writer.WriteField("apikey", apiKey); err != nil {
		return "", fmt.Errorf("failed to write API key: %v", err)
	}

	part, err := writer.CreateFormFile("file", "image.png")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := part.Write(imageData); err != nil {
		return "", fmt.Errorf("failed to write image data: %v", err)
	}
	writer.Close()

	// Make request
	req, err := http.NewRequest("POST", spaceURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result ocrResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if result.IsErroredOnProcessing {
		return "", fmt.Errorf("OCR API error: %v", result.ErrorMessage)
	}

	if len(result.ParsedResults) == 0 {
		return "", fmt.Errorf("no text found in image")
	}

	return result.ParsedResults[0].ParsedText, nil
}
func GoogleOCRText(imageData []byte) (string, error) {
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
