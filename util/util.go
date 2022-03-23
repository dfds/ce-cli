package util

import (
	"io/ioutil"
	"os"
)

func LoadJSONFileAsString(jsonFileName string) (string, error) {

	// if the file exists then...
	if _, err := os.Stat(jsonFileName); err == nil {

		// read in the JSON
		jsonFile, err := os.Open(jsonFileName)

		// in the case of an error just return it
		if err != nil {
			return "", err
		} else {
			// defer the closure of the file
			defer jsonFile.Close()

			// read the file into the byte array
			byteValue, _ := ioutil.ReadAll(jsonFile)

			// return the json byte array in string format
			return string(byteValue), err
		}
	} else {
		// file not found so return err
		return "", err
	}
}
