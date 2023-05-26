package usecase

import (
	"context"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/mdblp/go-common/clients/status"
	orcaSchema "github.com/mdblp/orca/schema"
	twV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/stretchr/testify/assert"
	"github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/infrastructure"
)

var (
	date1, _ = time.Parse(time.RFC3339Nano, "2021-09-01T00:00:00.000Z")
	date2, _ = time.Parse(time.RFC3339Nano, "2021-09-01T00:00:01.000Z")
	date3, _ = time.Parse(time.RFC3339Nano, "2021-09-01T00:05:00.000Z")
	date4, _ = time.Parse(time.RFC3339Nano, "2021-09-01T01:00:00.000Z")
	param1   = orcaSchema.CurrentParameter{
		Name:          "name1",
		Value:         "value1",
		Unit:          "unit1",
		Level:         1,
		EffectiveDate: &date1,
	}
	param2 = orcaSchema.CurrentParameter{
		Name:          "name2",
		Value:         "value2",
		Unit:          "unit2",
		Level:         2,
		EffectiveDate: &date2,
	}
	param3 = orcaSchema.CurrentParameter{
		Name:          "name3",
		Value:         "value3",
		Unit:          "unit3",
		Level:         1,
		EffectiveDate: &date4,
	}
	histoParam1 = orcaSchema.HistoryParameter{
		CurrentParameter: param1,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        date3,
		Timezone:         "UTC",
		TimezoneOffset:   0,
	}
	histoParam2 = orcaSchema.HistoryParameter{
		CurrentParameter: param2,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        date3,
		Timezone:         "UTC",
		TimezoneOffset:   0,
	}
	histoParam3 = orcaSchema.HistoryParameter{
		CurrentParameter: param3,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        date4,
		Timezone:         "UTC",
		TimezoneOffset:   0,
	}
)

func Test_groupByChangeDate(t *testing.T) {
	tests := []struct {
		name  string
		given []orcaSchema.HistoryParameter
		want  []GroupedHistoryParameters
	}{
		{
			name:  "should return empty array when empty array input",
			given: []orcaSchema.HistoryParameter{},
			want:  []GroupedHistoryParameters{},
		},
		{
			name:  "should return one group when one param input",
			given: []orcaSchema.HistoryParameter{histoParam1},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: date3,
					Parameters: []orcaSchema.HistoryParameter{histoParam1},
				},
			},
		},
		{
			name: "should return one group when two param with same timestamp input",
			given: []orcaSchema.HistoryParameter{
				histoParam1,
				histoParam2,
			},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: date3,
					Parameters: []orcaSchema.HistoryParameter{histoParam1, histoParam2},
				},
			},
		},
		{
			name: "should return two group when two param with same timestamp and third with different timestamp but same day",
			given: []orcaSchema.HistoryParameter{
				histoParam1,
				histoParam2,
				histoParam3,
			},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: date3,
					Parameters: []orcaSchema.HistoryParameter{histoParam1, histoParam2},
				},
				{
					ChangeDate: date4,
					Parameters: []orcaSchema.HistoryParameter{histoParam3},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupByChangeDate(tt.given)
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].ChangeDate.After(tt.want[j].ChangeDate) })
			sort.Slice(got, func(i, j int) bool { return got[i].ChangeDate.After(got[j].ChangeDate) })
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByChangeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

/*When no settings are found, we should not raise an error in getLatestPumpSettings*/
func TestAPI_getLatestPumpSettings_handleNotFound(t *testing.T) {
	/*Given*/
	token := "TestAPI_getLatestPumpSettings_token"
	userId := "TestAPI_getLatestPumpSettings_userId"
	ctx := context.Background()
	testLogger := log.New(os.Stdout, "realisticJsonInput", log.LstdFlags|log.Lshortfile)
	timeContext := common.TimeItContext(ctx)
	clientError := status.StatusError{
		Status: status.NewStatus(http.StatusNotFound, "GetSettings: no settings found"),
	}
	writer := writeFromIter{}
	mockRepository := infrastructure.NewMockPatientDataRepository()
	mockTideV2 := twV2Client.NewMock()
	usecase := NewPatientDataUseCase(testLogger, mockTideV2, mockRepository, true)
	mockTideV2.On("GetSettings", timeContext, userId, token).Return(nil, &clientError)

	/*When*/
	res, err := usecase.getLatestPumpSettings(timeContext, "traceId", userId, &writer, token)

	/*Then*/
	assert.Nil(t, err)
	assert.Nil(t, res)
}

func Test_convertToMgdl(t *testing.T) {
	tests := []struct {
		name  string
		given float64
		want  float64
	}{
		{"should handle positive value", 10, 180},
		{"should handle 0 value mmol", 0, 0},
		{"should handle positive value with decimal", 2.5, 45},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToMgdl(tt.given)
			assert.Equalf(t, tt.want, got, "convertToMgdl(%v)", tt.given)
		})
	}
}

func TestIsConvertibleUnit(t *testing.T) {
	tests := []struct {
		name     string
		given    string
		expected bool
	}{
		{"Valid Unit - mg/dL", MgdL, true},
		{"Valid Unit - mmol/L", MmolL, true},
		{"Invalid Unit - g/L", "g/L", false},
		{"Empty Unit", "", false},
		{"Random String", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConvertibleUnit(tt.given)
			if result != tt.expected {
				t.Errorf("isConvertibleUnit(%q) = %v, expected %v", tt.given, result, tt.expected)
			}
		})
	}
}

func Test_convertToMmol(t *testing.T) {
	tests := []struct {
		name  string
		given float64
		want  float64
	}{
		{"should handle 0 value", 0, 0},
		{"should handle positive value", 180, 10},
		{"should handle positive value with decimal", 45.1, 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToMmol(tt.given)
			assert.Equalf(t, tt.want, got, "convertToMmol(%v)", tt.given)
		})
	}
}
