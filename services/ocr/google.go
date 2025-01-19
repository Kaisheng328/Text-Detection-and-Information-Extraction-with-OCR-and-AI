package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	vision "cloud.google.com/go/vision/apiv1"
	"example.com/kaisheng/common/helper"
)

func GoogleOCRText(base64image string) (string, error) {
	ctx := context.Background()

	imageData, err := base64.StdEncoding.DecodeString(helper.Base64format(base64image))
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// googleCred := option.WithCredentialsFile(os.Getenv("GOOGLE_CRED"))
	client, err := vision.NewImageAnnotatorClient(ctx)
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
