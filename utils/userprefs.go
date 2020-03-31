package utils

type (
	UserPref struct {
		UserId   string
		VeryLow  float32
		Low      float32
		High     float32
		VeryHigh float32
	}
)

func GetUserPrefs(userIds []string) []UserPref {
	var out []UserPref
	out = make([]UserPref, len(userIds))
	for idx, usrId := range userIds {
		out[idx] = defaultUserPref(usrId)
	}
	return out
}
func defaultUserPref(usrId string) UserPref {
	return UserPref{
		UserId:   usrId,
		VeryLow:  3.0,
		Low:      3.9,
		High:     10.0,
		VeryHigh: 13.9,
	}
}
