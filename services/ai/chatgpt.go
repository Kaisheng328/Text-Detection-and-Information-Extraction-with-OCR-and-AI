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

func ProcessChatgptAI(formatedText string, modelname string) (string, error) {
	var fullResponse string
	ChatgptKey := os.Getenv("OPENAI_API_KEY")
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
	req, err := http.NewRequest("POST", enums.OpenaiURL, bytes.NewBuffer(jsonBody))
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
