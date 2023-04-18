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
	effDate1, _ = time.Parse(time.RFC3339Nano, "2021-09-01T00:00:00.000Z")
	effDate2, _ = time.Parse(time.RFC3339Nano, "2021-09-01T00:00:01.000Z")
	effDate3, _ = time.Parse(time.RFC3339Nano, "2021-09-02T00:00:00.000Z")
	param1      = orcaSchema.CurrentParameter{
		Name:          "name1",
		Value:         "value1",
		Unit:          "unit1",
		Level:         1,
		EffectiveDate: &effDate1,
	}
	param2 = orcaSchema.CurrentParameter{
		Name:          "name2",
		Value:         "value2",
		Unit:          "unit2",
		Level:         2,
		EffectiveDate: &effDate2,
	}
	param3 = orcaSchema.CurrentParameter{
		Name:          "name3",
		Value:         "value3",
		Unit:          "unit3",
		Level:         1,
		EffectiveDate: &effDate3,
	}
	histoParam1 = orcaSchema.HistoryParameter{
		CurrentParameter: param1,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        effDate2,
		Timezone:         "UTC",
		TimezoneOffset:   0,
	}
	histoParam2 = orcaSchema.HistoryParameter{
		CurrentParameter: param2,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        effDate2,
		Timezone:         "UTC",
		TimezoneOffset:   0,
	}
	histoParam3 = orcaSchema.HistoryParameter{
		CurrentParameter: param3,
		ChangeType:       "added",
		PreviousValue:    "",
		PreviousUnit:     "",
		Timestamp:        effDate3,
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
					ChangeDate: effDate2,
					Parameters: []orcaSchema.HistoryParameter{histoParam1},
				},
			},
		},
		{
			name: "should return one group when two param with same timestamp day input",
			given: []orcaSchema.HistoryParameter{
				histoParam1,
				histoParam2,
			},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: effDate2,
					Parameters: []orcaSchema.HistoryParameter{histoParam1, histoParam2},
				},
			},
		},
		{
			name: "should return two group when two param with same timestamp day and third with different day input",
			given: []orcaSchema.HistoryParameter{
				histoParam1,
				histoParam2,
				histoParam3,
			},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: effDate2,
					Parameters: []orcaSchema.HistoryParameter{histoParam1, histoParam2},
				},
				{
					ChangeDate: effDate3,
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
	testLogger := log.New(os.Stdout, "test", log.LstdFlags|log.Lshortfile)
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

func Test_getMgdl(t *testing.T) {
	tests := []struct {
		name  string
		given float64
		want  float64
	}{
		{
			name:  "should convert mmol to mgdl",
			given: 45,
			want:  25,
		},
		{
			name:  "should handle 0 value mmol",
			given: 0,
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMgdl(tt.given)
			assert.Equalf(t, tt.want, got, "getMgdl(%v)", tt.given)
		})
	}
}
