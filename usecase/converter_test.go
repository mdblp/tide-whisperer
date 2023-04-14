package usecase

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonToCsv(t *testing.T) {
	testCases := []struct {
		name            string
		jsonString      string
		expectedHeaders []string
		expectedRows    [][]string
		expectError     bool
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
					"address": {
						"city": "New York",
						"state": "NY"
					}
				},
				{
					"name": "Jane Smith",
					"email": "jane.smith@example.com",
					"address": {
						"city": "Los Angeles",
						"state": "CA",
						"country": "USA"
					}
				}
			]`,
			expectedHeaders: []string{"address.city", "address.country", "address.state", "age", "email", "name"},
			expectedRows: [][]string{
				{"New York", "", "NY", "30", "", "John Doe"},
				{"Los Angeles", "USA", "CA", "", "jane.smith@example.com", "Jane Smith"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := jsonToCsv(testCase.jsonString)
			if testCase.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				csvReader := csv.NewReader(bytes.NewReader(result.Bytes()))
				headers, _ := csvReader.Read()
				assert.Equal(t, testCase.expectedHeaders, headers)

				for _, expectedRow := range testCase.expectedRows {
					row, _ := csvReader.Read()
					assert.Equal(t, expectedRow, row)
				}
			}
		})
	}
}
