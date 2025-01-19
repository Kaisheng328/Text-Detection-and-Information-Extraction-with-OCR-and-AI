package ocr

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"example.com/kaisheng/common/enums"
	"example.com/kaisheng/common/helper"
	"google.golang.org/api/iterator"
)

// timestamp
type APIKey struct {
	ID                 string    `firestore:"-"`
	Key                string    `firestore:"key"`
	Balance            int       `firestore:"balance"`
	Usage              int       `firestore:"usage"`
	CreatedAt          time.Time `firestore:"createdAt"`
	ExpiresAt          time.Time `firestore:"expiredAt"`
	ParseImageEndpoint string    `firestore:"parseImageEndpoint"`
}

// Get Firestore client
func GetFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %v", err)
	}
	return client, nil
}

func getLeastUsedAPIKey(ctx context.Context, client *firestore.Client) (*APIKey, error) {
	// Define the usage limit

	query := client.Collection(enums.CollectionPath).
		Where("balance", ">", 0).
		Where("expiredAt", ">", time.Now()).
		OrderBy("expiredAt", firestore.Asc).
		Limit(1)

	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no available API keys with usage count > 0")
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch API key: %v", err)
	}

	var apiKey APIKey
	if err := doc.DataTo(&apiKey); err != nil {
		return nil, fmt.Errorf("failed to parse API key data: %v", err)
	}
	apiKey.ID = doc.Ref.ID
	return &apiKey, nil
}

func decrementUsageCount(ctx context.Context, client *firestore.Client, key string) error {
	docRef := client.Collection(enums.CollectionPath).Doc(key)
	if _, err := docRef.Update(ctx, []firestore.Update{{Path: "balance", Value: firestore.Increment(-1)}, {Path: "usage", Value: firestore.Increment(1)}}); err != nil {
		return fmt.Errorf("failed to decrement balance in transaction: %v", err)
	}
	return nil
}

// Perform OCR using the least used API key
func SpaceOCRText(base64image string) (string, error) {
	ctx := context.Background()
	client, err := GetFirestoreClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()
	if err != nil {
		log.Fatalf("Failed to fetch parseImageEndpoint from Firestore: %v", err)
	}

	// Fetch the least used API key
	apiKey, err := getLeastUsedAPIKey(ctx, client)
	if err != nil {
		return "", err
	}

	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(base64image, prefix) {
		base64image = prefix + base64image
	}

	// Use the selected API key for OCR
	config := helper.InitConfig(apiKey.Key, apiKey.ParseImageEndpoint, "eng", helper.OCREngine2)
	result, err := config.ParseFromBase64(base64image)
	if err != nil {
		return "", err
	}

	// Increment usage count in Firestore
	if err := decrementUsageCount(ctx, client, apiKey.ID); err != nil {
		log.Printf("Failed to decrement usage count for key %s: %v\n", apiKey.ID, err)
	}

	return result.JustText(), nil
}

//https://myapi.ocr.space/conversions post to check apikey count. more details visit https://ocr.space/ocrapi/myapi
