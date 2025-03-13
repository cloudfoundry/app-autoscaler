package configutil

import (
	"fmt"
	"os"
)

func MaterializeContentInFile(folderName, fileName, content string) (string, error) {
	dirPath := fmt.Sprintf("/tmp/%s", folderName)

	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

func MaterializeContentInTmpFile(folderName, fileName, content string) (string, error) {
	dirPath, err := os.MkdirTemp("", folderName)
	if err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}
