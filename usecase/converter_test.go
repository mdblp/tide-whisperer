package usecase

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var realisticJsonInput = `[
{"activeSchedule":"Normal","deviceId":"1234","id":"46f81417-cb19-4eec-8317-e0b0bf41046e","payload":{"basalsecurityprofile":null,"cgm":{"apiVersion":"v1","endOfLifeTransmitterDate":"2020-04-12T15:53:54Z","expirationDate":"2021-04-12T15:53:54Z","manufacturer":"Dexcom","name":"G6","swVersionTransmitter":"v1","transmitterId":"a1234"},"device":{"deviceId":"1234","imei":"1234567890","name":"DBLG1","manufacturer":"Diabeloop","swVersion":"beta"},"history":[{"changeDate":"2019-03-26T00:02:00Z","parameters":[{"name":"IOB_TAU_S","value":"78","unit":"min","level":2,"effectiveDate":"2019-03-26T00:02:00Z","changeType":"added","timestamp":"2019-03-26T00:02:00Z","timezone":"UTC"},{"name":"IOB_TAU_S","value":"90","unit":"min","level":2,"effectiveDate":"2019-03-26T00:04:00Z","changeType":"updated","previousValue":"78","previousUnit":"min","timestamp":"2019-03-26T00:04:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"102","unit":"%","level":1,"effectiveDate":"2019-03-26T00:02:00Z","changeType":"added","timestamp":"2019-03-26T00:02:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"80","unit":"%","level":1,"effectiveDate":"2019-03-26T00:04:00Z","changeType":"updated","previousValue":"102","previousUnit":"%","timestamp":"2019-03-26T00:04:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"123","unit":"%","level":1,"effectiveDate":"2019-03-26T00:02:00Z","changeType":"added","timestamp":"2019-03-26T00:02:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"99","unit":"%","level":1,"effectiveDate":"2019-03-26T00:04:00Z","changeType":"updated","previousValue":"123","previousUnit":"%","timestamp":"2019-03-26T00:04:00Z","timezone":"UTC"}]},{"changeDate":"2019-11-20T00:00:00Z","parameters":[{"name":"IOB_TAU_S","value":"75","unit":"min","level":2,"effectiveDate":"2019-11-20T00:00:00Z","changeType":"updated","previousValue":"90","previousUnit":"min","timestamp":"2019-11-20T00:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"100","unit":"%","level":1,"effectiveDate":"2019-11-20T00:00:00Z","changeType":"updated","previousValue":"80","previousUnit":"%","timestamp":"2019-11-20T00:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"100","unit":"%","level":1,"effectiveDate":"2019-11-20T00:00:00Z","changeType":"updated","previousValue":"99","previousUnit":"%","timestamp":"2019-11-20T00:00:00Z","timezone":"UTC"}]},{"changeDate":"2020-01-05T08:00:00Z","parameters":[{"name":"IOB_TAU_S","value":"80","unit":"min","level":2,"effectiveDate":"2020-01-05T08:00:00Z","changeType":"updated","previousValue":"75","previousUnit":"min","timestamp":"2020-01-05T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"110","unit":"%","level":1,"effectiveDate":"2020-01-05T08:00:00Z","changeType":"updated","previousValue":"100","previousUnit":"%","timestamp":"2020-01-05T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"110","unit":"%","level":1,"effectiveDate":"2020-01-05T08:00:00Z","changeType":"updated","previousValue":"100","previousUnit":"%","timestamp":"2020-01-05T08:00:00Z","timezone":"UTC"}]},{"changeDate":"2020-01-09T08:00:00Z","parameters":[{"name":"IOB_TAU_S","value":"90","unit":"min","level":2,"effectiveDate":"2020-01-09T08:00:00Z","changeType":"updated","previousValue":"80","previousUnit":"min","timestamp":"2020-01-09T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"120","unit":"%","level":1,"effectiveDate":"2020-01-09T08:00:00Z","changeType":"updated","previousValue":"110","previousUnit":"%","timestamp":"2020-01-09T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"120","unit":"%","level":1,"effectiveDate":"2020-01-09T08:00:00Z","changeType":"updated","previousValue":"110","previousUnit":"%","timestamp":"2020-01-09T08:00:00Z","timezone":"UTC"}]},{"changeDate":"2020-01-13T08:00:00Z","parameters":[{"name":"IOB_TAU_S","value":"85","unit":"min","level":2,"effectiveDate":"2020-01-13T08:00:00Z","changeType":"updated","previousValue":"90","previousUnit":"min","timestamp":"2020-01-13T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"110","unit":"%","level":1,"effectiveDate":"2020-01-13T08:00:00Z","changeType":"updated","previousValue":"120","previousUnit":"%","timestamp":"2020-01-13T08:00:00Z","timezone":"UTC"}]},{"changeDate":"2020-01-17T08:00:00Z","parameters":[{"name":"IOB_TAU_S","value":"75","unit":"min","level":2,"effectiveDate":"2020-01-17T08:00:00Z","changeType":"updated","previousValue":"85","previousUnit":"min","timestamp":"2020-01-17T08:00:00Z","timezone":"UTC"},{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"100","unit":"%","level":1,"effectiveDate":"2020-01-17T08:00:00Z","changeType":"updated","previousValue":"110","previousUnit":"%","timestamp":"2020-01-17T08:00:00Z","timezone":"UTC"}]}],"parameters":[{"name":"MEAL_RATIO_LUNCH_FACTOR","value":"100","unit":"%","level":1,"effectiveDate":"2020-01-17T08:00:00Z"},{"name":"MEAL_RATIO_DINNER_FACTOR","value":"120","unit":"%","level":1,"effectiveDate":"2020-01-17T08:00:00Z"},{"name":"IOB_TAU_S","value":"75","unit":"min","level":2,"effectiveDate":"2020-01-17T08:00:00Z"}],"pump":{"expirationDate":"2021-04-12T15:53:54Z","manufacturer":"VICENTRA","name":"Kaleido","swVersion":"beta","serialNumber":"123456"}},"time":"2020-01-17T08:00:00Z","timezone":"UTC","type":"pumpSettings","uploadId":"bed6c7bf-db15-411d-9412-fac675c1e7ff"},
{"id":"bb49b132-266c-4fbc-aac7-19fbf8bc9a27","lastUpdateDate":"2019-03-26T00:04:00Z","level":2,"name":"IOB_TAU_S","previousValue":"78","subType":"deviceParameter","time":"2019-03-26T00:04:00Z","timezone":"UTC","type":"deviceEvent","units":"min","uploadId":"3f6ae721-b48e-4820-b65a-dca77903bffb","value":"90"},
{"id":"e10eb6f1-e824-4046-a3bc-be8cfe23e114","lastUpdateDate":"2019-03-26T00:02:00Z","level":1,"name":"MEAL_RATIO_DINNER_FACTOR","subType":"deviceParameter","time":"2019-03-26T00:02:00Z","timezone":"UTC","type":"deviceEvent","units":"%","uploadId":"fb2a7ddc-9835-4fd1-ae2c-c633e7a64e3d","value":"102"},
{"id":"e67f01a9-c543-4359-bd5a-5df87c29a3bc","lastUpdateDate":"2019-03-26T00:04:00Z","level":1,"name":"MEAL_RATIO_DINNER_FACTOR","previousValue":"102","subType":"deviceParameter","time":"2019-03-26T00:04:00Z","timezone":"UTC","type":"deviceEvent","units":"%","uploadId":"e98877ee-812f-4027-931d-0e0d5db047d5","value":"80"},
{"id":"608e1fcf-a9d2-472c-b650-1ba8453fff31","lastUpdateDate":"2020-01-13T08:00:00Z","level":2,"name":"IOB_TAU_S","previousValue":"90","subType":"deviceParameter","time":"2020-01-13T08:00:00Z","timezone":"UTC","type":"deviceEvent","units":"min","uploadId":"25728402-b31a-43f8-b1f3-3791348d2b44","value":"85"},
{"id":"f0c5e955-b5fc-4e39-974b-77e0e493d07a","lastUpdateDate":"2020-01-05T08:00:00Z","level":1,"name":"MEAL_RATIO_LUNCH_FACTOR","previousValue":"110","subType":"deviceParameter","time":"2020-01-17T08:00:00Z","timezone":"UTC","type":"deviceEvent","units":"%","uploadId":"a99db5b3-cb6f-4eee-8bf9-515dcffce77d","value":"100"},
{"id":"c8059a4e77927230d23dcb8f0f5ce345","normal":5,"prescriptor":"auto","subType":"normal","time":"2020-01-20T18:00:00Z","timezone":"Europe/Paris","type":"bolus","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"6397f90b7c38aefc9761b9d1fa4852bb","normal":0.58064526,"prescriptor":"auto","subType":"normal","time":"2020-01-20T06:00:00Z","timezone":"Europe/Paris","type":"bolus","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"duration":{"units":"hours","value":2},"eventId":"confidential_0","guid":"confidential_0","id":"1fa85ce87de050900e20176b022c9e13","inputTime":"2020-01-20T18:26:00Z","subType":"confidential","time":"2020-01-20T18:31:00Z","timezone":"Europe/Paris","type":"deviceEvent","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"duration":{"units":"hours","value":2},"eventId":"zen_0","guid":"zen_0","id":"9d035ba687ea3d636c0350ce55445c27","inputTime":"2020-01-20T08:55:00Z","subType":"zen","time":"2020-01-20T09:00:00Z","timezone":"Europe/Paris","type":"deviceEvent","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"d830c1e5932eb14fdbd520e4c2e2f848","subType":"reservoirChange","time":"2020-01-19T10:00:00Z","timezone":"Europe/Paris","type":"deviceEvent","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"996238a4cf4b814b5eaeb23b10b492ee","subType":"reservoirChange","time":"2020-01-07T10:00:00Z","timezone":"Europe/Paris","type":"deviceEvent","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"e9d8fba091310be650986b6547503506","meal":"rescuecarbs","nutrition":{"carbohydrate":{"net":15,"units":"grams"}},"time":"2020-01-20T12:00:00Z","timezone":"Europe/Paris","type":"food","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"ce8cd73cf6dd591d4159b6a50b66bdba","meal":"rescuecarbs","nutrition":{"carbohydrate":{"net":5,"units":"grams"}},"time":"2020-01-04T00:00:00Z","timezone":"Europe/Paris","type":"food","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"duration":{"units":"seconds","value":1800},"eventId":"pa_18","guid":"pa_18","id":"8e9cecdcf40db4e97246e7060165cf72","reportedIntensity":"medium","time":"2020-01-20T04:00:00Z","timezone":"Europe/Paris","type":"physicalActivity","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"duration":{"units":"seconds","value":3600},"eventId":"pa_3","guid":"pa_3","id":"e37bd9b9216f0a6af72f4e4853eb10fc","reportedIntensity":"medium","time":"2020-01-04T13:00:00Z","timezone":"Europe/Paris","type":"physicalActivity","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"bolus":"6209584f31cf9587bea2b33b29964256","carbInput":35,"id":"6c18e61bf25a86ff00577b504b5b9899","inputMeal":{"fat":"yes"},"inputTime":"2022-01-01T07:58:00Z","time":"2020-01-20T12:00:00Z","timezone":"Europe/Paris","type":"wizard","units":"mmol/L","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"bolus":"05a6c48f3a37e067af1f190bf643a4f8","carbInput":40,"id":"0a463fa0cc5d794a1232e126190ae120","inputMeal":{"fat":"yes"},"inputTime":"2022-01-01T07:58:00Z","time":"2020-01-04T00:00:00Z","timezone":"Europe/Paris","type":"wizard","units":"mmol/L","uploadId":"eb3865714c16b536d88dbd3c831955ab"},
{"id":"cbg_f8dfcabca768_2020-01-20_0","time":"2020-01-20T12:00:00Z","timezone":"Europe/Paris","type":"cbg","units":"mmol/L","value":10.5},
{"id":"cbg_f8dfcabca768_2020-01-09_0","time":"2020-01-09T16:00:00Z","timezone":"Europe/Paris","type":"cbg","units":"mmol/L","value":10.5},
{"deliveryType":"automated","duration":600000,"id":"basal_f8dfcabca768_2020-01-20_71","rate":0.8,"time":"2020-01-20T11:50:00Z","timezone":"Europe/Paris","type":"basal"},
{"deliveryType":"automated","duration":72000000,"id":"basal_f8dfcabca768_2020-01-05_0","rate":1,"time":"2020-01-05T04:00:00Z","timezone":"Europe/Paris","type":"basal"},
{"_dataState":"open","_deduplicator":{"name":"org.tidepool.deduplicator.none","version":"1.0.0"},"_state":"open","client":{"name":"api.dev.diabeloop.com","version":"1.0.0"},"dataSetType":"continuous","deviceManufacturers":["Diabeloop"],"deviceModel":"DBLG1","deviceTags":["cgm","insulin-pump"],"id":"eb3865714c16b536d88dbd3c831955ab","revision":1,"time":"2019-11-20T00:00:00Z","timezone":"America/New_York","type":"upload","uploadId":"eb3865714c16b536d88dbd3c831955ab","version":"1.0.0"}]
`

func TestJsonToCsv(t *testing.T) {
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
			expectedHeaders: []string{"_dataState", "_deduplicator.name", "_deduplicator.version", "_state", "activeSchedule", "bolus", "carbInput", "client.name", "client.version", "dataSetType", "deliveryType", "deviceId", "deviceManufacturers", "deviceModel", "deviceTags", "duration", "duration.units", "duration.value", "eventId", "guid", "id", "inputMeal.fat", "inputTime", "lastUpdateDate", "level", "meal", "name", "normal", "nutrition.carbohydrate.net", "nutrition.carbohydrate.units", "payload.basalsecurityprofile", "payload.cgm.apiVersion", "payload.cgm.endOfLifeTransmitterDate", "payload.cgm.expirationDate", "payload.cgm.manufacturer", "payload.cgm.name", "payload.cgm.swVersionTransmitter", "payload.cgm.transmitterId", "payload.device.deviceId", "payload.device.imei", "payload.device.manufacturer", "payload.device.name", "payload.device.swVersion", "payload.history", "payload.parameters", "payload.pump.expirationDate", "payload.pump.manufacturer", "payload.pump.name", "payload.pump.serialNumber", "payload.pump.swVersion", "prescriptor", "previousValue", "rate", "reportedIntensity", "revision", "subType", "time", "timezone", "type", "units", "uploadId", "value", "version"},
			expectedRows: [][]string{
				{"", "", "", "", "Normal", "", "", "", "", "", "", "1234", "", "", "", "", "", "", "", "", "46f81417-cb19-4eec-8317-e0b0bf41046e", "", "", "", "", "", "", "", "", "", "", "v1", "2020-04-12T15:53:54Z", "2021-04-12T15:53:54Z", "Dexcom", "G6", "v1", "a1234", "1234", "1234567890", "Diabeloop", "DBLG1", "beta", "[map[changeDate:2019-03-26T00:02:00Z parameters:[map[changeType:added effectiveDate:2019-03-26T00:02:00Z level:2 name:IOB_TAU_S timestamp:2019-03-26T00:02:00Z timezone:UTC unit:min value:78] map[changeType:updated effectiveDate:2019-03-26T00:04:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:78 timestamp:2019-03-26T00:04:00Z timezone:UTC unit:min value:90] map[changeType:added effectiveDate:2019-03-26T00:02:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR timestamp:2019-03-26T00:02:00Z timezone:UTC unit:% value:102] map[changeType:updated effectiveDate:2019-03-26T00:04:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR previousUnit:% previousValue:102 timestamp:2019-03-26T00:04:00Z timezone:UTC unit:% value:80] map[changeType:added effectiveDate:2019-03-26T00:02:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR timestamp:2019-03-26T00:02:00Z timezone:UTC unit:% value:123] map[changeType:updated effectiveDate:2019-03-26T00:04:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:123 timestamp:2019-03-26T00:04:00Z timezone:UTC unit:% value:99]]] map[changeDate:2019-11-20T00:00:00Z parameters:[map[changeType:updated effectiveDate:2019-11-20T00:00:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:90 timestamp:2019-11-20T00:00:00Z timezone:UTC unit:min value:75] map[changeType:updated effectiveDate:2019-11-20T00:00:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR previousUnit:% previousValue:80 timestamp:2019-11-20T00:00:00Z timezone:UTC unit:% value:100] map[changeType:updated effectiveDate:2019-11-20T00:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:99 timestamp:2019-11-20T00:00:00Z timezone:UTC unit:% value:100]]] map[changeDate:2020-01-05T08:00:00Z parameters:[map[changeType:updated effectiveDate:2020-01-05T08:00:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:75 timestamp:2020-01-05T08:00:00Z timezone:UTC unit:min value:80] map[changeType:updated effectiveDate:2020-01-05T08:00:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR previousUnit:% previousValue:100 timestamp:2020-01-05T08:00:00Z timezone:UTC unit:% value:110] map[changeType:updated effectiveDate:2020-01-05T08:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:100 timestamp:2020-01-05T08:00:00Z timezone:UTC unit:% value:110]]] map[changeDate:2020-01-09T08:00:00Z parameters:[map[changeType:updated effectiveDate:2020-01-09T08:00:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:80 timestamp:2020-01-09T08:00:00Z timezone:UTC unit:min value:90] map[changeType:updated effectiveDate:2020-01-09T08:00:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR previousUnit:% previousValue:110 timestamp:2020-01-09T08:00:00Z timezone:UTC unit:% value:120] map[changeType:updated effectiveDate:2020-01-09T08:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:110 timestamp:2020-01-09T08:00:00Z timezone:UTC unit:% value:120]]] map[changeDate:2020-01-13T08:00:00Z parameters:[map[changeType:updated effectiveDate:2020-01-13T08:00:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:90 timestamp:2020-01-13T08:00:00Z timezone:UTC unit:min value:85] map[changeType:updated effectiveDate:2020-01-13T08:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:120 timestamp:2020-01-13T08:00:00Z timezone:UTC unit:% value:110]]] map[changeDate:2020-01-17T08:00:00Z parameters:[map[changeType:updated effectiveDate:2020-01-17T08:00:00Z level:2 name:IOB_TAU_S previousUnit:min previousValue:85 timestamp:2020-01-17T08:00:00Z timezone:UTC unit:min value:75] map[changeType:updated effectiveDate:2020-01-17T08:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR previousUnit:% previousValue:110 timestamp:2020-01-17T08:00:00Z timezone:UTC unit:% value:100]]]]", "[map[effectiveDate:2020-01-17T08:00:00Z level:1 name:MEAL_RATIO_LUNCH_FACTOR unit:% value:100] map[effectiveDate:2020-01-17T08:00:00Z level:1 name:MEAL_RATIO_DINNER_FACTOR unit:% value:120] map[effectiveDate:2020-01-17T08:00:00Z level:2 name:IOB_TAU_S unit:min value:75]]", "2021-04-12T15:53:54Z", "VICENTRA", "Kaleido", "123456", "beta", "", "", "", "", "", "", "2020-01-17T08:00:00Z", "UTC", "pumpSettings", "", "bed6c7bf-db15-411d-9412-fac675c1e7ff", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "bb49b132-266c-4fbc-aac7-19fbf8bc9a27", "", "", "2019-03-26T00:04:00Z", "2", "", "IOB_TAU_S", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "78", "", "", "", "deviceParameter", "2019-03-26T00:04:00Z", "UTC", "deviceEvent", "min", "3f6ae721-b48e-4820-b65a-dca77903bffb", "90", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "e10eb6f1-e824-4046-a3bc-be8cfe23e114", "", "", "2019-03-26T00:02:00Z", "1", "", "MEAL_RATIO_DINNER_FACTOR", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "deviceParameter", "2019-03-26T00:02:00Z", "UTC", "deviceEvent", "%", "fb2a7ddc-9835-4fd1-ae2c-c633e7a64e3d", "102", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "e67f01a9-c543-4359-bd5a-5df87c29a3bc", "", "", "2019-03-26T00:04:00Z", "1", "", "MEAL_RATIO_DINNER_FACTOR", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "102", "", "", "", "deviceParameter", "2019-03-26T00:04:00Z", "UTC", "deviceEvent", "%", "e98877ee-812f-4027-931d-0e0d5db047d5", "80", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "608e1fcf-a9d2-472c-b650-1ba8453fff31", "", "", "2020-01-13T08:00:00Z", "2", "", "IOB_TAU_S", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "90", "", "", "", "deviceParameter", "2020-01-13T08:00:00Z", "UTC", "deviceEvent", "min", "25728402-b31a-43f8-b1f3-3791348d2b44", "85", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "f0c5e955-b5fc-4e39-974b-77e0e493d07a", "", "", "2020-01-05T08:00:00Z", "1", "", "MEAL_RATIO_LUNCH_FACTOR", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "110", "", "", "", "deviceParameter", "2020-01-17T08:00:00Z", "UTC", "deviceEvent", "%", "a99db5b3-cb6f-4eee-8bf9-515dcffce77d", "100", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "c8059a4e77927230d23dcb8f0f5ce345", "", "", "", "", "", "", "5", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "auto", "", "", "", "", "normal", "2020-01-20T18:00:00Z", "Europe/Paris", "bolus", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "6397f90b7c38aefc9761b9d1fa4852bb", "", "", "", "", "", "", "0.58064526", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "auto", "", "", "", "", "normal", "2020-01-20T06:00:00Z", "Europe/Paris", "bolus", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "hours", "2", "confidential_0", "confidential_0", "1fa85ce87de050900e20176b022c9e13", "", "2020-01-20T18:26:00Z", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "confidential", "2020-01-20T18:31:00Z", "Europe/Paris", "deviceEvent", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "hours", "2", "zen_0", "zen_0", "9d035ba687ea3d636c0350ce55445c27", "", "2020-01-20T08:55:00Z", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "zen", "2020-01-20T09:00:00Z", "Europe/Paris", "deviceEvent", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "d830c1e5932eb14fdbd520e4c2e2f848", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "reservoirChange", "2020-01-19T10:00:00Z", "Europe/Paris", "deviceEvent", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "996238a4cf4b814b5eaeb23b10b492ee", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "reservoirChange", "2020-01-07T10:00:00Z", "Europe/Paris", "deviceEvent", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "e9d8fba091310be650986b6547503506", "", "", "", "", "rescuecarbs", "", "", "15", "grams", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-20T12:00:00Z", "Europe/Paris", "food", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "ce8cd73cf6dd591d4159b6a50b66bdba", "", "", "", "", "rescuecarbs", "", "", "5", "grams", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-04T00:00:00Z", "Europe/Paris", "food", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "seconds", "1800", "pa_18", "pa_18", "8e9cecdcf40db4e97246e7060165cf72", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "medium", "", "", "2020-01-20T04:00:00Z", "Europe/Paris", "physicalActivity", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "seconds", "3600", "pa_3", "pa_3", "e37bd9b9216f0a6af72f4e4853eb10fc", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "medium", "", "", "2020-01-04T13:00:00Z", "Europe/Paris", "physicalActivity", "", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "6209584f31cf9587bea2b33b29964256", "35", "", "", "", "", "", "", "", "", "", "", "", "", "", "6c18e61bf25a86ff00577b504b5b9899", "yes", "2022-01-01T07:58:00Z", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-20T12:00:00Z", "Europe/Paris", "wizard", "mmol/L", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "05a6c48f3a37e067af1f190bf643a4f8", "40", "", "", "", "", "", "", "", "", "", "", "", "", "", "0a463fa0cc5d794a1232e126190ae120", "yes", "2022-01-01T07:58:00Z", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-04T00:00:00Z", "Europe/Paris", "wizard", "mmol/L", "eb3865714c16b536d88dbd3c831955ab", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "cbg_f8dfcabca768_2020-01-20_0", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-20T12:00:00Z", "Europe/Paris", "cbg", "mmol/L", "", "10.5", ""},
				{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "cbg_f8dfcabca768_2020-01-09_0", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "2020-01-09T16:00:00Z", "Europe/Paris", "cbg", "mmol/L", "", "10.5", ""},
				{"", "", "", "", "", "", "", "", "", "", "automated", "", "", "", "", "600000", "", "", "", "", "basal_f8dfcabca768_2020-01-20_71", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "0.8", "", "", "", "2020-01-20T11:50:00Z", "Europe/Paris", "basal", "", "", "", ""},
				{"", "", "", "", "", "", "", "", "", "", "automated", "", "", "", "", "7.2e+07", "", "", "", "", "basal_f8dfcabca768_2020-01-05_0", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "1", "", "", "", "2020-01-05T04:00:00Z", "Europe/Paris", "basal", "", "", "", ""},
				{"open", "org.tidepool.deduplicator.none", "1.0.0", "open", "", "", "", "api.dev.diabeloop.com", "1.0.0", "continuous", "", "", "[Diabeloop]", "DBLG1", "[cgm insulin-pump]", "", "", "", "", "", "eb3865714c16b536d88dbd3c831955ab", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "1", "", "2019-11-20T00:00:00Z", "America/New_York", "upload", "", "eb3865714c16b536d88dbd3c831955ab", "", "1.0.0"}},
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
