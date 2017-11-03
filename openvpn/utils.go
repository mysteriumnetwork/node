package openvpn

import (
	"math/rand"
	"os"
	"time"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()),
)

const finenameCharset = "1234567890abcdefghijklmnopqrstuvwxyz"

func tempFilename(directory, filePrefix, fileExtension string) (filePath string) {
	for i := 0; i < 10000; i++ {
		filePath = directory + "/" + filePrefix + randomString(10, finenameCharset) + fileExtension
		if _, err := os.Stat(filePath); os.IsExist(err) {
			continue
		}
	}
	return
}

func randomString(size int, charset string) string {
	filename := make([]byte, size)
	for i := range filename {
		charsetIndex := seededRand.Intn(len(charset))
		filename[i] = charset[charsetIndex]
	}

	return string(filename)
}
