package aggregates

import (
	"encoding/json"
	"github.com/tidepool-org/tide-whisperer/store"
	"github.com/tidepool-org/tide-whisperer/utils"
	"log"
	"net/http"
	"strings"
)

func TimeInRange(res http.ResponseWriter, req *http.Request, storage store.Storage) {
	log.Printf("----------------> TimeInRange API")
	q := req.URL.Query()
	userIds := strings.Split(q.Get("userIds"), ",")
	userPrefs := utils.GetUserPrefs(userIds)

	iter := storage.GetTimeInRangeData(userPrefs)
	defer iter.Close()

	var writeCount int

	res.Header().Add("Content-Type", "application/json")
	res.Write([]byte("["))

	var results map[string]interface{}
	for iter.Next(&results) {
		if len(results) > 0 {
			if bytes, err := json.Marshal(results); err != nil {
				log.Printf(" Marshal returned error: %s", err)
			} else {
				if writeCount > 0 {
					res.Write([]byte(","))
				}
				res.Write([]byte("\n"))
				res.Write(bytes)
				writeCount += 1
			}
		}
	}

	if writeCount > 0 {
		res.Write([]byte("\n"))
	}
	res.Write([]byte("]"))
}
