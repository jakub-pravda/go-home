package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mu sync.Mutex
var lastTemperatures = make(map[string]int)

type TimeTable struct {
	Start       int64 `json:"start"` // seconds from midnight
	End         int64 `json:"end"`   // seconds from midnight
	Temperature int   `json:"temperature"`
}

type TemperatureScheduler struct {
	Topic              string      `json:"topic"`
	DefaultTemperature int         `json:"defaultTemperature"`
	TimeTable          []TimeTable `json:"timeTable"`
}

// Check if time table is in defined interval
func timeTableInInterval(timeTable TimeTable, time time.Time) bool {
	timeSeconds := getSecondsFromMidnight(time.Hour(), time.Minute())

	if timeTable.Start == 0 && timeTable.End == 0 { // all day
		return true
	} else if timeTable.Start > timeTable.End { // eg. 22:00 - 06:00
		return timeSeconds >= timeTable.Start && timeSeconds <= 86400 || timeSeconds >= 0 && timeSeconds <= timeTable.End
	} else { // eg. 06:00 - 22:00
		return timeSeconds >= timeTable.Start && timeSeconds <= timeTable.End
	}
}

// Get configured temperature for given time
func getTemperatureAtTime(scheduler TemperatureScheduler, time time.Time) int {
	for _, timeTable := range scheduler.TimeTable {
		if timeTableInInterval(timeTable, time) {
			return timeTable.Temperature
		}
	}
	return scheduler.DefaultTemperature
}

// Check if temperature update is needed
//
//	in: scheduler - temperature scheduler table
//	out: bool - true if update is needed; tempature - temperature to set (0 if update not needed)
func temperatureUpdateNeeded(scheduler TemperatureScheduler, time time.Time) (bool, int) {
	mu.Lock()
	defer mu.Unlock()

	temperature := getTemperatureAtTime(scheduler, time)
	lastTemperature, exist := lastTemperatures[scheduler.Topic]
	if exist {
		if lastTemperature != temperature {
			log.Printf("Temperature update needed for %s, last: %d, current: %d", scheduler.Topic, lastTemperature, temperature)
			lastTemperatures[scheduler.Topic] = temperature
			return true, temperature
		} else {
			return false, 0
		}
	} else {
		log.Printf("Creating temperature state for topic %s, value: %d", scheduler.Topic, temperature)
		lastTemperatures[scheduler.Topic] = temperature
		return false, 0
	}
}

/*
	 	Parse json configuration to the TemperatureScheduler struct
		Input example:
		```json
		{
			"topic": "myhome-kr/livingroom/danfoss-thermo-01",
			"defaultTemperature": 22,
			"timeTable": [
				{
					"start": "22:30",
					"end": "05:30",
					"temperature": 18
				}
			]
		}`
		```

		output struct:
		```
		{myhome-kr/livingroom/danfoss-thermo-01 22 [{2230 530 18}]}
		```
*/
func parseTimeTable(tempSchedulerJson string) (TemperatureScheduler, error) {
	log.Printf("Parsing time table: %s", tempSchedulerJson)
	var tempScheduler TemperatureScheduler
	err := json.Unmarshal([]byte(tempSchedulerJson), &tempScheduler)
	if err != nil {
		log.Fatalf("Time table parsing failed")
		return TemperatureScheduler{}, err
	}
	return tempScheduler, nil
}

func getSecondsFromMidnight(hours int, minutes int) int64 {
	date := time.Date(1970, 1, 1, hours, minutes, 0, 0, time.UTC)
	return date.Unix()
}

// convert string to integer, fatal when conversion fails
func convertOrFatal(strInt string) int {
	result, err := strconv.Atoi(strInt)
	if err != nil {
		log.Fatalf("Can't parse integer: %s", strInt)
	}
	return result
}

func (t *TimeTable) UnmarshalJSON(data []byte) error {
	// custom unmarshaler for TimeTable
	var dat map[string]interface{}

	if err := json.Unmarshal(data, &dat); err != nil {
		return err
	}

	startSplit := strings.Split(dat["start"].(string), ":")
	startSeconds := getSecondsFromMidnight(convertOrFatal(startSplit[0]), convertOrFatal(startSplit[1]))

	endSplit := strings.Split(dat["end"].(string), ":")
	endSeconds := getSecondsFromMidnight(convertOrFatal(endSplit[0]), convertOrFatal(endSplit[1]))

	t.Start = startSeconds
	t.End = endSeconds
	t.Temperature = int(dat["temperature"].(float64))

	return nil
}
