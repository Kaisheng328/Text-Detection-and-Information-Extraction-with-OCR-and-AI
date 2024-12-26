# OCR AI Cloud Service

## Overview

OCR AI Cloud Service is a cloud-based application designed to extract text from images using Optical Character Recognition (OCR) technology. Leveraging artificial intelligence, it analyzes the extracted text to retrieve detailed information, such as names, addresses, document types, and other relevant data. This service streamlines data processing for applications like visitor management systems, identity verification, and more.

## Features

- **OCR Functionality:** Extract text from various types of images, including licenses, identification cards, and contractor passes.
- **AI-Powered Insights:** Analyze extracted text to obtain structured information, such as names, addresses, document types, and countries.
- **Scalable Cloud Deployment:** Built for high availability and scalability using modern cloud infrastructure.

## Technologies Used

- **Backend:** Golang
- **Cloud Services:** Google Cloud Functions
- **AI Integration:** ChatGPT and Ollama AI
- **Deployment:** Railway and Google Cloud Run

## Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/Kaisheng328/Text-Detection-and-Information-Extraction-with-OCR-and-AI.git
   cd Text-Detection-and-Information-Extraction-with-OCR-and-AI

## Sending a Request with Postman

1. Open Postman and create a new request.
2. Set the request type to `POST` and enter the following URL: https://ocridentity-s5m47uwooa-as.a.run.app/ic
3. Go to the "Body" tab and select "raw" as the data format.
4. Choose "JSON" from the dropdown.
5. Add the following JSON payload in the request body:
```json
{
  "ocr_provider": "ocr-google",
  "ai_provider": "chatgpt",
  "ai_model": "",
  "content": "/9j/4AAQSkZJRgABAQEAYABgAAD/..."
}
```
