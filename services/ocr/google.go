package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"

	vision "cloud.google.com/go/vision/apiv1"
	"example.com/kaisheng/common/helper"
	"google.golang.org/api/option"
)

func GoogleOCRText(base64image string) (string, error) {
	ctx := context.Background()

	imageData, err := base64.StdEncoding.DecodeString(helper.Base64format(base64image))
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

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

// func GosseractOCRText(base64image string) (string, error) {
// 	// Decode base64 image
// 	if strings.HasPrefix(base64image, "data:image/") {
// 		commaIndex := strings.Index(base64image, ",")
// 		if commaIndex != -1 {
// 			base64image = base64image[commaIndex+1:]
// 		} else {
// 			return "", fmt.Errorf("invalid base64 image format")
// 		}
// 	}
// 	imageData, err := base64.StdEncoding.DecodeString(base64image)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to decode base64 image: %v", err)
// 	}
// 	// tmpFile, err := os.CreateTemp("", "image-*.png")
// 	// if err != nil {
// 	// 	return "", fmt.Errorf("failed to create temp file: %v", err)
// 	// }
// 	// defer os.Remove(tmpFile.Name())
// 	// _, err = tmpFile.Write(imageData)
// 	// if err != nil {
// 	// 	return "", fmt.Errorf("failed to write image data to temp file: %v", err)
// 	// }
// 	// tmpFile.Close()
// 	// preprocessedPath, err := preprocessImage(tmpFile.Name())
// 	// if err != nil {
// 	// 	return "", fmt.Errorf("failed to preprocess image: %v", err)
// 	// }
// 	// defer os.Remove(preprocessedPath)

// 	// Perform OCR using gosseract
// 	client := gosseract.NewClient()
// 	defer client.Close()

// 	client.SetImageFromBytes(imageData)
// 	text, err := client.Text()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to perform OCR: %v", err)
// 	}

// 	return text, nil
// }

// func preprocessImage(inputPath string) (string, error) {
// 	// Load the image
// 	img := gocv.IMRead(inputPath, gocv.IMReadColor)
// 	if img.Empty() {
// 		return "", fmt.Errorf("failed to read image")
// 	}
// 	defer img.Close()

// 	// Convert to grayscale
// 	grayImg := gocv.NewMat()
// 	defer grayImg.Close()
// 	gocv.CvtColor(img, &grayImg, gocv.ColorBGRToGray)

// 	// Apply Gaussian Blur to reduce noise
// 	blurredImg := gocv.NewMat()
// 	defer blurredImg.Close()
// 	gocv.GaussianBlur(grayImg, &blurredImg, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

// 	// Apply binary threshold to convert to black and white
// 	bwImg := gocv.NewMat()
// 	defer bwImg.Close()
// 	gocv.Threshold(blurredImg, &bwImg, 128, 255, gocv.ThresholdBinary)

// 	// Save the preprocessed image to a temporary file
// 	outputPath := inputPath + "_preprocessed.png"
// 	if ok := gocv.IMWrite(outputPath, bwImg); !ok {
// 		return "", fmt.Errorf("failed to write preprocessed image")
// 	}

// 	return outputPath, nil
// }
// case "Gosseract":
// 	result, err := GosseractOCRText(base64image)
// 	if err != nil {
// 		log.Printf("Error in GosseractOCRText: %v", err)
// 		return "", err
// 	}
// 	return result, nil
