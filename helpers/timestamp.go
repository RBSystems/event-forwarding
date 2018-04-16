package helpers

import (
	"time"

	"github.com/byuoitav/event-forwarding/logger"
)

type CrestronEvent struct {
	Type         string `json:"type"`
	Timestamp    string `json:"timestamp"`
	NewTimestamp string `json:"@timestamp"`
	EventTime    string `json:"eventTime"`
	EventDate    string `json:"eventDate"`
	Device       struct {
		Hostname    string `json:"hostname"`
		Description string `json:"description"`
		IPAddress   string `json:"ipAddress"`
		MacAddress  string `json:"macAddress"`
	} `json:"device"`
	Room struct {
		Building    string `json:"building"`
		RoomNumber  string `json:"roomNumber"`
		Coordinates string `json:"coordinates"`
		Floor       string `json:"floor"`
	} `json:"room"`
	Action struct {
		Actor       string `json:"actor"`
		Description string `json:"description"`
	} `json:"action"`
	Session string `json:"session"`
}

func ConvertTimestamp(e CrestronEvent) CrestronEvent {

	//parse the old timestamp
	//2018-4-12T09:40:57-0600
	t, err := time.Parse("2006-1-02T15:04:05-0700", e.Timestamp)
	if err != nil {
		logger.L.Infof("Couldn't parse time: %v", err.Error())
		//couldn't be parsed...
		return e
	}

	//set the new timestamp
	e.NewTimestamp = t.Format(time.RFC3339)
	return e
}
