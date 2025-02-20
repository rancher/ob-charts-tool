package util

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func GetHTTPBody(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return body, err
	}
	return body, nil
}
