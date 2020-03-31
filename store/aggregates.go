package store

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	// For debug only pretty printing requests
	"encoding/json"
	"github.com/tidepool-org/tide-whisperer/utils"
	"log"
	"time"
)

type (
	userDate struct {
		minDate time.Time
		maxDate time.Time
	}
)

func (d MongoStoreClient) getMaxDate(userIds []string) map[string]userDate {
	query := []bson.M{}
	matchCbgDataForUserIds := bson.M{
		"type":    "cbg",
		"_userId": bson.M{"$in": userIds},
	}
	query = append(query, bson.M{"$match": matchCbgDataForUserIds})
	groupQuery := bson.M{
		"_id":     "$_userId",
		"maxDate": bson.M{"$max": "$time"},
	}
	query = append(query, bson.M{"$group": groupQuery})
	projectQuery := bson.M{
		"_id":     "$_id",
		"maxDate": dateFromString("$maxDate"),
	}
	query = append(query, bson.M{"$project": projectQuery})
	session := d.session.Copy()
	var res []map[string]interface{}
	err := mgoDataCollection(session).Pipe(query).All(&res)
	// For debug purpose pretty printing mongo requests:
	if bytes, err := json.MarshalIndent(query, "", "    "); err != nil {
		log.Printf("json marshalled value error %s", err)
	} else {
		log.Printf("getMaxDate request : %s", bytes)
	}
	var out map[string]userDate
	out = make(map[string]userDate)
	if err == nil {
		for _, res := range res {
			max := res["maxDate"].(time.Time)
			min := max.Add(-(time.Hour * 24))
			out[res["_id"].(string)] = userDate{
				maxDate: max,
				minDate: min,
			}
		}
	}
	return out
}

func dateFromString(field string) bson.M {
	return bson.M{"$dateFromString": bson.M{"dateString": field}}
}

func formatUserPrefs(userPrefs []utils.UserPref) ([]string, []bson.M) {
	var userIds []string
	userIds = make([]string, len(userPrefs))
	var bsonPrefs []bson.M
	bsonPrefs = make([]bson.M, len(userPrefs))
	for idx, pref := range userPrefs {
		userIds[idx] = pref.UserId
		bsonPrefs[idx] = bson.M{
			"veryLow":     pref.VeryLow,
			"low":         pref.Low,
			"high":        pref.High,
			"veryHigh":    pref.VeryHigh,
			"cgmInterval": 5,
		}
	}
	return userIds, bsonPrefs
}

func generateAggregateQuery(userIds []string, bsonPrefs []bson.M, userDates map[string]userDate) []bson.M {

	finalQuery := []bson.M{}
	/*
		1st step of pipeline $match:
			filtering cbg data for userIds
			with time between max time and max time - 1 day
			(these dates are retrieved in a first step aggregate see getMaxDate)
	*/
	orDates := []bson.M{}
	for _, usrId := range userIds {
		dates := userDates[usrId]
		andUsr := []bson.M{
			{"_userId": usrId},
			{"time": bson.M{"$gte": dates.minDate.Format(time.RFC3339)}},
			{"time": bson.M{"$lte": dates.maxDate.Format(time.RFC3339)}},
		}
		orDates = append(orDates, bson.M{"$and": andUsr})
	}
	matchCbgDataForUserIds := bson.M{
		"$and": []bson.M{
			{"type": "cbg"},
			{"_userId": bson.M{"$in": userIds}},
			{"$or": orDates},
		},
	}
	finalQuery = append(finalQuery, bson.M{"$match": matchCbgDataForUserIds})

	/*
		2nd step of pipeline $project:
			projecting  only used fields i.e userId/time/value
			casting string time to real Date
			processing time - 1 day (for future filtering)
			joining data with user preferences (passed as parameter) i.e
				veryLow threshold
				low threshold
				high threshold
				veryHigh threshold
				cgmInterval
	*/
	castDate := dateFromString("$time")
	projectPrefsQuery := bson.M{
		"userId":        "$_userId",
		"time":          castDate,
		"timeYesterday": bson.M{"$add": []interface{}{castDate, -86400000}},
		"value":         "$value",
		"cbgBounds": bson.M{
			"$let": bson.M{
				"vars": bson.M{
					"userIds":   userIds,
					"allBounds": bsonPrefs,
				},
				"in": bson.M{
					"$arrayElemAt": []interface{}{
						"$$allBounds",
						bson.M{"$indexOfArray": []string{"$$userIds", "$_userId"}},
					},
				},
			},
		},
	}
	finalQuery = append(finalQuery, bson.M{"$project": projectPrefsQuery})
	/*
		3rd step of pipeline $project:
			projecting cbg category based on the value and threshold
	*/
	projectGroupCbgQuery := bson.M{
		"userId":        "$userId",
		"time":          "$time",
		"timeYesterday": "$timeYesterday",
		"cgmInterval":   "$cbgBounds.cgmInterval",
		"cbgCategory": bson.M{
			"$switch": bson.M{
				"branches": []bson.M{
					{
						"case": bson.M{"$lt": []string{"$value", "$cbgBounds.veryLow"}},
						"then": "veryLow",
					},
					{
						"case": bson.M{
							"$and": []bson.M{
								{"$gte": []string{"$value", "$cbgBounds.veryLow"}},
								{"$lt": []string{"$value", "$cbgBounds.low"}},
							},
						},
						"then": "low",
					},
					{
						"case": bson.M{
							"$and": []bson.M{
								{"$gt": []string{"$value", "$cbgBounds.high"}},
								{"$lte": []string{"$value", "$cbgBounds.veryHigh"}},
							},
						},
						"then": "high",
					},
					{
						"case": bson.M{"$gt": []string{"$value", "$cbgBounds.veryHigh"}},
						"then": "veryHigh",
					},
				},
				"default": "target",
			},
		},
	}
	finalQuery = append(finalQuery, bson.M{"$project": projectGroupCbgQuery})

	/*
		4th step of pipeline $group:
			grouping data by userId/category/cgmInterval  with:
				max time for all categories (lastCbgTime)
				count of veryLow category
				count of low category
				count of target category
				count of high category
				count of veryHigh category
				max time of veryLow category
				max time of low category
				max time of target category
				max time of high category
				max time of veryHigh category
	*/
	countQuery := bson.M{
		"_id": bson.M{
			"userId":      "$userId",
			"category":    "$cbgCategory",
			"cgmInterval": "$cgmInterval",
		},
		"lastCbgTime": bson.M{"$max": "$time"},
	}
	thresholdNames := []string{"veryLow", "low", "target", "high", "veryHigh"}

	for _, threshold := range thresholdNames {
		countQuery[threshold+"Count"] = bson.M{
			"$sum": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{
							"case": bson.M{"$eq": []string{"$cbgCategory", threshold}},
							"then": 1,
						},
					},
					"default": 0,
				},
			},
		}
		countQuery[threshold+"Time"] = bson.M{
			"$max": bson.M{
				"$switch": bson.M{
					"branches": []bson.M{
						{
							"case": bson.M{"$eq": []string{"$cbgCategory", threshold}},
							"then": "$time",
						},
					},
					"default": nil,
				},
			},
		}
	}

	finalQuery = append(finalQuery, bson.M{"$group": countQuery})
	/*
		5th step of pipeline $group:
			grouping data by userId:
				max of cgmInterval (already max unique value per userId)
				max of lastCbgTime (already a max unique value per userId)
				max of veryLow category (only one line per userId as a value <> 0)
				max of low category (only one line per userId as a value <> 0)
				max of target category (only one line per userId as a value <> 0)
				max of high category (only one line per userId as a value <> 0)
				max of veryHigh category (only one line per userId as a value <> 0)
				max time of veryLow category (only one line per userId as a value <> nil)
				max time of low category (only one line per userId as a value <> nil)
				max time of target category (only one line per userId as a value <> nil)
				max time of high category (only one line per userId as a value <> nil)
				max time of veryHigh category (only one line per userId as a value <> nil)
	*/
	finalGroupQuery := bson.M{
		"_id":         "$_id.userId",
		"lastCbgTime": bson.M{"$max": "$lastCbgTime"},
		"cgmInterval": bson.M{"$max": "$_id.cgmInterval"},
	}
	for _, threshold := range thresholdNames {
		finalGroupQuery[threshold+"Count"] = bson.M{
			"$max": ("$" + threshold + "Count"),
		}
		finalGroupQuery[threshold+"Time"] = bson.M{
			"$max": ("$" + threshold + "Time"),
		}
	}
	finalQuery = append(finalQuery, bson.M{"$group": finalGroupQuery})
	/*
		6th step of pipeline $projecting:
			projecting data for output with following structure:
				userId
				lastCbgTime
				count : // number of cbg per category //
					veryLow
					low
					target
					high
					veryHigh
				lastTime :  // max of cbg time per category //
					veryLow
					low
					target
					high
					veryHigh
				rate : // percentage of (cbg events)/ (total events) per category //
					veryLow
					low
					target
					high
					veryHigh
				totalTime: // (number of cbg) * (cgm time interval) per catgeory //
					veryLow
					low
					target
					high
					veryHigh
	*/
	finalProjectQuery := bson.M{
		"_id":         0,
		"userId":      "$_id",
		"lastCbgTime": "$lastCbgTime",
		"cgmInterval": "$cgmInterval",
	}
	counts := bson.M{}
	lastTimes := bson.M{}
	rates := bson.M{}
	totalTimes := bson.M{}
	fieldsTotal := make([]string, len(thresholdNames))
	for idx, threshold := range thresholdNames {
		counts[threshold] = "$" + threshold + "Count"
		lastTimes[threshold] = "$" + threshold + "Time"
		rates[threshold] = bson.M{
			"$multiply": []interface{}{
				bson.M{"$divide": []string{"$" + threshold + "Count", "$$total"}},
				100,
			},
		}
		totalTimes[threshold] = bson.M{
			"$multiply": []string{"$" + threshold + "Count", "$cgmInterval"},
		}
		fieldsTotal[idx] = "$" + threshold + "Count"
	}
	finalProjectQuery["count"] = counts
	finalProjectQuery["lastTime"] = lastTimes
	finalProjectQuery["rate"] = bson.M{
		"$let": bson.M{
			"vars": bson.M{
				"total": bson.M{"$add": fieldsTotal},
			},
			"in": rates,
		},
	}
	finalProjectQuery["totalTime"] = totalTimes
	finalQuery = append(finalQuery, bson.M{"$project": finalProjectQuery})

	// For debug purpose pretty printing mongo requests:
	if bytes, err := json.MarshalIndent(finalQuery, "", "    "); err != nil {
		log.Printf("json marshalled value error %s", err)
	} else {
		log.Printf("TIR aggregate request : %s", bytes)
	}
	return finalQuery
}

func (d MongoStoreClient) GetTimeInRangeData(userPrefs []utils.UserPref) StorageIterator {
	userIds, bsonPrefs := formatUserPrefs(userPrefs)
	session := d.session.Copy()
	dates := d.getMaxDate(userIds)
	var iter *mgo.Iter
	pipe := mgoDataCollection(session).Pipe(generateAggregateQuery(userIds, bsonPrefs, dates))
	iter = pipe.Iter()

	return &ClosingSessionIterator{session, iter}
}
