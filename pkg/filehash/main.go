package filehash

import (
	"hash/crc32"
	"fmt"
	"os"
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

	tablePolynomial := crc32.MakeTable(crc32.IEEE)
	hash := crc32.Checksum([]byte(fileContent), tablePolynomial)

	return float64(hash), nil
}
