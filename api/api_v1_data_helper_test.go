package api

import (
	orcaSchema "github.com/mdblp/orca/schema"
	"reflect"
	"sort"
	"testing"
	"time"
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
	type args struct {
		parameters []orcaSchema.HistoryParameter
	}
	tests := []struct {
		name string
		args args
		want []GroupedHistoryParameters
	}{
		{
			name: "should return empty array when empty array input",
			args: args{
				[]orcaSchema.HistoryParameter{},
			},
			want: []GroupedHistoryParameters{},
		},
		{
			name: "should return one group when one param input",
			args: args{
				[]orcaSchema.HistoryParameter{
					histoParam1,
				},
			},
			want: []GroupedHistoryParameters{
				{
					ChangeDate: effDate2,
					Parameters: []orcaSchema.HistoryParameter{histoParam1},
				},
			},
		},
		{
			name: "should return one group when two param with same timestamp day input",
			args: args{
				[]orcaSchema.HistoryParameter{
					histoParam1,
					histoParam2,
				},
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
			args: args{
				[]orcaSchema.HistoryParameter{
					histoParam1,
					histoParam2,
					histoParam3,
				},
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
			got := groupByChangeDate(tt.args.parameters)
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].ChangeDate.After(tt.want[j].ChangeDate) })
			sort.Slice(got, func(i, j int) bool { return got[i].ChangeDate.After(got[j].ChangeDate) })
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByChangeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
