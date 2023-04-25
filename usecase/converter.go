package usecase

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

func jsonToCsv(jsonString string) (*bytes.Buffer, error) {
	var jsonObjects []map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &jsonObjects)
	if err != nil {
		var singleJsonObject map[string]interface{}
		err2 := json.Unmarshal([]byte(jsonString), &singleJsonObject)
		if err2 != nil {
			return nil, errors.New("failed to unmarshal input JSON")
		}
		jsonObjects = []map[string]interface{}{singleJsonObject}
	}

	headersMap := make(map[string]struct{})
	for _, jsonObject := range jsonObjects {
		extractedHeaders, err := extractHeaders(jsonObject)
		if err != nil {
			return nil, err
		}
		for _, header := range extractedHeaders {
			headersMap[header] = struct{}{}
		}
	}

	headers := make([]string, 0, len(headersMap))
	for header := range headersMap {
		headers = append(headers, header)
	}
	sort.Strings(headers)

	csvBuffer := &bytes.Buffer{}
	csvWriter := csv.NewWriter(csvBuffer)
	csvWriter.Write(headers)
	csvWriter.Flush()

	for _, jsonObject := range jsonObjects {
		if err := writeCsvRow(jsonObject, headers, csvWriter); err != nil {
			return nil, err
		}
	}

	return csvBuffer, nil
}

func extractHeaders(jsonObject map[string]interface{}) ([]string, error) {
	headers := make([]string, 0)
	for key, value := range jsonObject {
		switch v := value.(type) {
		case map[string]interface{}:
			subHeaders, err := extractHeaders(v)
			if err != nil {
				return nil, err
			}
			for _, subHeader := range subHeaders {
				headers = append(headers, fmt.Sprintf("%s.%s", key, subHeader))
			}
		default:
			headers = append(headers, key)
		}
	}
	return headers, nil
}

func writeCsvRow(jsonObject map[string]interface{}, headers []string, csvWriter *csv.Writer) error {
	row := make([]string, len(headers))
	for i, header := range headers {
		parts := strings.Split(header, ".")
		value, err := getValue(jsonObject, parts)
		if err != nil {
			return err
		}
		row[i] = fmt.Sprintf("%v", value)
	}
	csvWriter.Write(row)
	csvWriter.Flush()
	return nil
}

func getValue(jsonObject map[string]interface{}, parts []string) (interface{}, error) {
	value, ok := jsonObject[parts[0]]
	/*json null value will be nil in go ... for the moment write nothing for this*/
	if !ok || value == nil {
		return "", nil // Return an empty string for missing keys
	}
	if len(parts) == 1 {
		/*This is thrown in the case where we're looking for duration but the current object
		is having a duration field which is an object with for example unit and value*/
		switch value.(type) {
		case map[string]interface{}:
			return "", nil // Return an empty string for missing keys
		default:
			return value, nil
		}
	}
	subObject, ok := value.(map[string]interface{})
	if !ok {
		/*This is thrown in the case where we're looking for duration.unit but the current object
		is having a duration field which is a number for example*/
		return "", nil // Return an empty string for missing keys
	}
	return getValue(subObject, parts[1:])
}
