package enums

const (
	OpenaiURL              = "https://api.openai.com/v1/chat/completions"
	SpaceURL               = "https://api.ocr.space/parse/image"
	Default_provider       = "ocr-google"
	Default_prompt_message = "This is Data Retrieve from GoogleVisionOCR. PLs help me to find {type: nric | passport | driving-license | visa | contractor pass | others , number:,name:,country: {code: ,name: }, address: {zip: , state: , city: , full: },  return result as **JSON (JavaScript Object Notation)** and must in stringify Json format make it machine readable message  dont use ```json!! If passport/visa/contractor got passport number, use the passpart number as number. If NRIC, driving lisence or other, find Idendity number to put as number. No explaination or further questions needed !!!"
)
