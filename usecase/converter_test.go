package usecase

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonToCsvError(t *testing.T) {
	tests := []struct {
		name       string
		jsonString string
	}{
		{
			name:       "incorrect JSON",
			jsonString: `test`,
		},
		{
			name:       "incorrect JSON",
			jsonString: `{test=""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jsonToCsv(tt.jsonString)
			assert.Error(t, err)
		})
	}
}

func TestJsonToCsv(t *testing.T) {
	realisticJsonInput, err := readConverterTestJsonFile()
	if err != nil {
		t.Fatal(err)
	}
	realisticCsvHeaders, realisticCsvRows, err := extractCsvFile()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name            string
		jsonString      string
		expectedHeaders []string
		expectedRows    [][]string
	}{
		{
			name: "Simple JSON",
			jsonString: `{
				"name": "John Doe",
				"age": 30,
				"city": "New York"
			}`,
			expectedHeaders: []string{"age", "city", "name"},
			expectedRows:    [][]string{{"30", "New York", "John Doe"}},
		},
		{
			name: "JSON with sub-object",
			jsonString: `{
				"name": "John Doe",
				"age": 30,
				"address": {
					"city": "New York",
					"state": "NY"
				}
			}`,
			expectedHeaders: []string{"address.city", "address.state", "age", "name"},
			expectedRows:    [][]string{{"New York", "NY", "30", "John Doe"}},
		},
		{
			name: "JSON with nested objects and different keys",
			jsonString: `{
				"name": "John Doe",
				"age": 30,
				"address": {
					"city": "New York",
					"state": "NY"
				},
				"education": {
					"degree": "Bachelor's",
					"major": "Computer Science"
				}
			}`,
			expectedHeaders: []string{"address.city", "address.state", "age", "education.degree", "education.major", "name"},
			expectedRows:    [][]string{{"New York", "NY", "30", "Bachelor's", "Computer Science", "John Doe"}},
		},
		{
			name: "JSON array with different keys and nested objects",
			jsonString: `[
				{
					"name": "John Doe",
					"age": 30,
					"address": "inline address, could messup with nested address"
				},
				{
					"name": "Jane Smith",
					"email": "jane.smith@example.com",
					"address": {
						"city": "Los Angeles",
						"state": "CA",
						"country": "USA"
					}
				},
				{
					"random": "object"
				}
			]`,
			expectedHeaders: []string{"address", "address.city", "address.country", "address.state", "age", "email", "name", "random"},
			expectedRows: [][]string{
				{"inline address, could messup with nested address", "", "", "", "30", "", "John Doe", ""},
				{"", "Los Angeles", "USA", "CA", "", "jane.smith@example.com", "Jane Smith", ""},
				{"", "", "", "", "", "", "", "object"},
			},
		},
		{
			name:            "should convert with success realistic data",
			jsonString:      realisticJsonInput,
			expectedHeaders: realisticCsvHeaders,
			expectedRows:    realisticCsvRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonToCsv(tt.jsonString)
			assert.NoError(t, err)
			csvReader := csv.NewReader(bytes.NewReader(result.Bytes()))
			headers, _ := csvReader.Read()
			assert.Equal(t, tt.expectedHeaders, headers)

			csvOutput, err := csvReader.ReadAll()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRows, csvOutput)
		})
	}
}

func readConverterTestJsonFile() (string, error) {
	jsonFile, err := openFile("./converter_test.json")
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return "", fmt.Errorf("cannot read JSON converter test file: %v", err)
	}
	return string(jsonBytes), nil
}

func extractCsvFile() ([]string, [][]string, error) {
	csvFile, err := openFile("./converter_test.csv")
	if err != nil {
		return nil, nil, err
	}
	defer csvFile.Close()
	// Read the CSV file using a CSV reader
	csvReader := csv.NewReader(csvFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read CSV converter test file: %v", err)
	}
	return records[0], records[1:], nil
}

func openFile(name string) (*os.File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("cannot open CSV converter test file: %v", err)
	}
	return file, err
}
