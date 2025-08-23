package parser

import (
	"encoding/xml"
	"io"
	"net/http"
	"rsshub/internal/models"
)

func ParseFeed(url string) (*models.RSS, error) {
	var rssparsed models.RSS
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respRead, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(respRead, &rssparsed)
	if err != nil {
		return nil, err
	}

	return &rssparsed, nil
}
