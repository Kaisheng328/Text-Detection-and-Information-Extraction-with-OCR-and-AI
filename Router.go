package router

import (
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"example.com/kaisheng/common/enums"
	"example.com/kaisheng/services/ai"
	"example.com/kaisheng/services/ocr"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
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
	functions.HTTP("ocrIdentity", ocrIdentity)
}

func ocrIdentity(w http.ResponseWriter, r *http.Request) {
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
		fmt.Printf("OCR error: %v", err)
		http.Error(w, `{"error": "Failed to perform OCR"}`, http.StatusInternalServerError)
		return
	}

	// Defualt responseBody
	responseBody := map[string]interface{}{
		"raw":       formatedText,
		"provider":  requestBody.OCRProvider,
		"result":    map[string]interface{}{},
		"rawResult": "",
	}

	if requestBody.AIProvider == "" {
		generateResponse(w, responseBody)
		return
	}

	models, providerExists := availableProviders[requestBody.AIProvider]
	if !providerExists {
		fmt.Printf("Invalid AI provider: %s", requestBody.AIProvider)
		generateResponse(w, responseBody)
		return
	}

	if requestBody.AIModel == "" {
		requestBody.AIModel = models[0]
	}

	// Process the extracted text with AI
	aiResponse, err := ProcessAI(formatedText, requestBody.AIProvider, requestBody.AIModel)
	if err != nil {
		generateResponse(w, responseBody)
		return
	}
	responseBody["rawResult"] = aiResponse

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &result); err != nil {
		// Log the decoding error
		fmt.Printf("Failed to decode aiResponse: %v\n", err)

		// Respond with only the OCR response
		generateResponse(w, responseBody)
		return
	}

	responseBody["result"] = result
	generateResponse(w, responseBody)
}

func ProcessAI(formatedText string, providerName string, modelName string) (string, error) {
	var aiResponse string
	var err error
	switch providerName {
	case "chatgpt":
		aiResponse, err = ai.ProcessChatgptAI(formatedText, modelName) // Call ChatGPT API function
		if err != nil {
			fmt.Printf("Error processing chatgpt AI: %v\n", err)
			return "", fmt.Errorf("error processing chatgpt AI: %v", err)
		}
	case "gemma":
		aiResponse, err = ai.ProcessGemmaAI(formatedText, modelName) // Call Gemma API function
		if err != nil {
			fmt.Printf("Error processing gemma AI: %v\n", err)
			return "", fmt.Errorf("error processing Gemma AI: %v", err)
		}
	default:
		fmt.Printf("Unknown provider: %s\n", providerName)
		aiResponse, err = ai.ProcessGemmaAI(formatedText, modelName) // Default to Gemma
		if err != nil {
			fmt.Printf("Error processing default AI: %v\n", err)
			return "", fmt.Errorf("error processing default AI: %v", err)
		}
	}
	return aiResponse, nil
}

func OCRVersion(content string, provider string) (string, error) {

	if provider == "" {
		provider = enums.Default_provider
	}
	switch provider {
	case enums.GOOGLE_CLOUD_PLATFORM:
		result, err := ocr.GoogleOCRText(content)
		if err != nil {
			return "", fmt.Errorf("Error in GoogleOCRText: %v", err)
		}
		return result, nil
	case enums.OCR_SPACE:
		result, err := ocr.SpaceOCRText(content)
		if err != nil {
			return "", fmt.Errorf("Error in SpaceOCRText: %v", err)
		}
		return result, nil
	default:
		return "", fmt.Errorf("unsupported OCR provider: %s", provider)
	}
}

func generateResponse(w http.ResponseWriter, responseBody map[string]interface{}) {
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
	}
}
