package util

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
