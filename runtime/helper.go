package runtime

import (
	"encoding/json"
	"net/http"
)

func JSONRequest(url string, value interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&value)
	if err != nil {
		return err
	}
	return nil
}
