package helper

import (
	"strings"
)

func Base64format(base64image string) string {
	b64 := ""
	if strings.HasPrefix(base64image, "data:image/") {
		commaIndex := strings.Index(base64image, ",")
		if commaIndex != -1 {
			base64image = base64image[commaIndex+1:]
		} else {
			return b64
		}
	}
	return base64image
}
