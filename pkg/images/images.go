package images

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

var validExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

type ImageInfo struct {
	Base64Data string
	Filename   string
}

func GetRandomBase64Images(dirPath string, count int) ([]ImageInfo, error) {
	rand.Seed(time.Now().UnixNano())

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	var validFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if validExtensions[ext] {
			validFiles = append(validFiles, filepath.Join(dirPath, file.Name()))
		}
	}

	if len(validFiles) == 0 {
		return nil, fmt.Errorf("no valid images found in directory")
	}

	// Seleccionar imágenes aleatorias sin repetición
	numToSelect := count
	if numToSelect > len(validFiles) {
		numToSelect = len(validFiles)
	}

	indices := rand.Perm(len(validFiles))[:numToSelect]
	var selectedImages []string
	for _, idx := range indices {
		selectedImages = append(selectedImages, validFiles[idx])
	}

	var imageInfos []ImageInfo
	for _, imgPath := range selectedImages {
		data, err := ioutil.ReadFile(imgPath)
		if err != nil {
			return nil, fmt.Errorf("error reading image file %s: %w", imgPath, err)
		}

		mimeType := "image/jpeg"
		ext := strings.ToLower(filepath.Ext(imgPath))
		if ext == ".png" {
			mimeType = "image/png"
		}

		base64Str := base64.StdEncoding.EncodeToString(data)
		imageInfos = append(imageInfos, ImageInfo{
			Base64Data: fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str),
			Filename:   filepath.Base(imgPath),
		})
	}

	return imageInfos, nil
}
