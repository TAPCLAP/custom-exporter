package filehash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
)

type FileHash struct {
	File string
	Hash float64
}

func Calculate(filePath string) (float64, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("error reading file: %v", err)
	}

	hash := sha256.Sum256(fileContent)

	hashString := hex.EncodeToString(hash[:])

	lastTenChars := hashString[len(hashString)-10:]

	number, err := strconv.ParseUint(lastTenChars, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting to number: %v", err)
	}

	return float64(number), nil
}
