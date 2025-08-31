package parser

import (
"encoding/xml"
"io"
"net/http"
"rsshub/internal/domain"
)

// RSSParser реализует интерфейс domain.RSSParser
type RSSParser struct{}

// NewRSSParser создает новый экземпляр RSSParser
func NewRSSParser() *RSSParser {
return &RSSParser{}
}

// ParseFeed выполняет HTTP-запрос по URL и парсит RSS
func (p *RSSParser) ParseFeed(url string) (*domain.RSS, error) {
var rssparsed domain.RSS
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
