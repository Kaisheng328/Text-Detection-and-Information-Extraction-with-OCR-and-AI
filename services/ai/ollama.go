package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"example.com/kaisheng/common/enums"
)

func ProcessGemmaAI(formatedText string, modelname string) (string, error) {
	var fullResponse string
	host := os.Getenv("OLLAMA_HOST")
	api := os.Getenv("OLLAMA_API")
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	OllamaKey := os.Getenv("OLLAMA_KEY")
	ollamaURL := fmt.Sprintf("http://%s/%s/%s", host, api, endpoint)
	prompt := fmt.Sprintf("%s, %s", enums.Default_prompt_message, formatedText)
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
