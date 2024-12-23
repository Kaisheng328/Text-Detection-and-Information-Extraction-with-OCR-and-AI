package router

import (
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"

	"example.com/kaisheng/common/enums"
	"example.com/kaisheng/services/ai"
	"example.com/kaisheng/services/ocr"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	_ "github.com/go-sql-driver/mysql"
)

type OCRInput struct {
	Content     string `json:"content"`
	OCRProvider string `json:"ocr_provider"`
	AIProvider  string `json:"ai_provider"`
	AIModel     string `json:"ai_model"`
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

	formatedText, err := OCRVersion(requestBody.Content, requestBody.OCRProvider)
	if err != nil {
		log.Printf("OCR error: %v", err)
		http.Error(w, `{"error": "Failed to perform OCR"}`, http.StatusInternalServerError)
		return
	}

	if requestBody.AIProvider == "" && requestBody.AIModel == "" {
		responseBody := map[string]interface{}{
			"ocrResponse": map[string]interface{}{
				"text":     formatedText,
				"provider": requestBody.OCRProvider,
			},
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}
	models, providerExists := availableProviders[requestBody.AIProvider]
	if !providerExists {
		log.Printf("Invalid AI provider: %s", requestBody.AIProvider)
		responseBody := map[string]interface{}{
			"ocrResponse": map[string]interface{}{
				"text":     formatedText,
				"provider": requestBody.OCRProvider,
			},
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}
	if requestBody.AIModel == "" {
		requestBody.AIModel = models[0]
	}
	// Process the extracted text with AI
	aiResponse, err := ProcessAI(formatedText, requestBody.AIProvider, requestBody.AIModel)
	if err != nil {
		responseBody := map[string]interface{}{
			"ocrResponse": map[string]interface{}{
				"text":     formatedText,
				"provider": requestBody.OCRProvider,
			},
		}
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}

	// Respond with the AI response
	responseBody := map[string]interface{}{
		"ocrResponse": map[string]interface{}{
			"text":     formatedText,
			"provider": requestBody.OCRProvider,
		},
	}
	var aiResponseData map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &aiResponseData); err != nil {
		// Log the decoding error
		log.Printf("Failed to decode aiResponse: %v\n", err)

		// Respond with only the OCR response
		if err := json.NewEncoder(w).Encode(responseBody); err != nil {
			http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		}
		return
	}
	responseBody["aiResponse"] = aiResponseData
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
	}
}

func ProcessAI(formatedText string, providerName string, modelName string) (string, error) {
	var aiResponse string
	var err error
	switch providerName {
	case "chatgpt":
		aiResponse, err = ai.ProcessChatgptAI(formatedText, modelName) // Call ChatGPT API function
		if err != nil {
			log.Printf("Error processing chatgpt AI: %v\n", err)
			return "", fmt.Errorf("error processing chatgpt AI: %v", err)
		}
	case "gemma":
		aiResponse, err = ai.ProcessGemmaAI(formatedText, modelName) // Call Gemma API function
		if err != nil {
			log.Printf("Error processing gemma AI: %v\n", err)
			return "", fmt.Errorf("error processing Gemma AI: %v", err)
		}
	default:
		log.Printf("Unknown provider: %s\n", providerName)
		aiResponse, err = ai.ProcessGemmaAI(formatedText, modelName) // Default to Gemma
		if err != nil {
			log.Printf("Error processing default AI: %v\n", err)
			return "", fmt.Errorf("error processing default AI: %v", err)
		}
	}
	return aiResponse, nil
}

func OCRVersion(Content string, provider string) (string, error) {

	if provider == "" {
		provider = enums.Default_provider
	}
	switch provider {
	case "ocr-google":
		result, err := ocr.GoogleOCRText(Content)
		if err != nil {
			log.Printf("Error in GoogleOCRText: %v", err)
			return "", err
		}
		return result, nil
	case "ocr-space":
		result, err := ocr.SpaceOCRText(Content)
		if err != nil {
			log.Printf("Error in SpaceOCRText: %v", err)
			return "", err
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported OCR provider: %s", provider)
	}
}
