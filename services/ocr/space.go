package ocr

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	ocr "github.com/ranghetto/go_ocr_space"
)

var currentKeyIndex int
var apiKeys []string

func initial() {
	apiKeys = loadAPIKeys()
	currentKeyIndex = 0
}
func loadAPIKeys() []string {

	keys := os.Getenv("SPACE_CRED")
	if keys == "" {
		fmt.Println("No API keys found. Set SPACE_CREDS in your environment.")
		os.Exit(1)
	}
	return SplitKeys(keys)
}

func getNextAPIKey() (string, error) {
	if len(apiKeys) == 0 {
		return "", errors.New("no API keys available")
	}
	key := apiKeys[currentKeyIndex]
	currentKeyIndex = (currentKeyIndex + 1) % len(apiKeys)
	return key, nil
}

func SpaceOCRText(base64image string) (string, error) {
	initial()
	for i := 0; i < len(apiKeys); i++ {
		spaceAPIKey, err := getNextAPIKey()
		if err != nil {
			return "", err
		}

		config := ocr.InitConfig(spaceAPIKey, "eng", ocr.OCREngine2)
		result, err := config.ParseFromBase64(base64image)
		if err != nil {
			if strings.Contains(err.Error(), "Quota exceeded") {
				log.Printf("Quota exceeded for API key %s: %v\n", spaceAPIKey, err)
				continue
			}

			// For other errors, log and stop
			fmt.Printf("Error with API key %s: %v\n", spaceAPIKey, err)
			return "", err
		}

		return result.JustText(), nil
	}

	return "", errors.New("all API keys failed")
}

func SplitKeys(keys string) []string {
	keyList := strings.Split(keys, ",")
	for i := range keyList {
		keyList[i] = strings.TrimSpace(keyList[i])
	}
	return keyList
}
