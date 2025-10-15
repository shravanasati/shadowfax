package response

import (
	"crypto/sha1"
	"fmt"
)

func sha1Hash(data []byte) [sha1.Size]byte {
	return sha1.Sum(data)
}

func prepareEtagValue(val string) string {
	return fmt.Sprintf(`"%x"`, sha1Hash([]byte(val)))
}
