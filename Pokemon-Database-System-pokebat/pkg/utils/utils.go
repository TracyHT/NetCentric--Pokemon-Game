package utils

import (
	"encoding/json"
	"io/ioutil"
)

func SaveToFile(filePath string, data interface{}) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, file, 0644)
}

func LoadFromFile(filePath string, data interface{}) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, data)
}
