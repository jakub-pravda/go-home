package main

import (
	"log"
	"testing"
	"time"
)

func TestIfTimeTableParsedCorrectly(t *testing.T) {
	testData := `
	{
		"topic": "myhome-kr/livingroom/danfoss-thermo-01",
		"defaultTemperature": 22,
		"timeTable": [
			{
				"start": "22:00",
				"end": "06:00",
				"temperature": 18
			}
		]
	}`

	// TODO expected

	result, err := parseTimeTable(testData)

	log.Printf("Result: %v", result)

	if err != nil {
		log.Fatalf(err.Error())
	}
}

func TestIfTemperatureAtTimeReturnsValidValue(t *testing.T) {
	testTimeTables := []TimeTable{
		{
			Start:       55800, // 15:30
			End:         72000, // 20:00
			Temperature: 25,
		},
		{
			Start:       79200, // 22:00
			End:         18000, // 05:00
			Temperature: 18,
		},
	}
	testScheduler := TemperatureScheduler{
		Topic:              "myhome-kr/livingroom/danfoss-thermo-01",
		DefaultTemperature: 22,
		TimeTable:          testTimeTables,
	}

	testTimesWithExpected := map[time.Time]int{
		time.Date(2023, 2, 3, 16, 0, 0, 0, time.UTC): 25,
		time.Date(2023, 2, 3, 21, 0, 0, 0, time.UTC): 22,
		time.Date(2023, 2, 4, 2, 15, 0, 0, time.UTC): 18,
		time.Date(2023, 2, 4, 6, 30, 0, 0, time.UTC): 22,
	}

	for testTime, expected := range testTimesWithExpected {
		result := getTemperatureAtTime(testScheduler, testTime)
		if result != expected {
			log.Fatalf("getTemperatureAtTime failed, expected: %d, got: %d", expected, result)
		}
	}
}

func TestTimeTableInInterval(t *testing.T) {
	testTime := time.Date(2023, 2, 4, 12, 0, 0, 0, time.UTC) // 12:00
	testData := [4]TimeTable{
		{
			Start: 36000, // 10:00
			End:   72000, // 20:00
		},
		{
			Start: 72000, // 20:00
			End:   46800, // 13:00
		},
		{
			Start: 0, // 00:00
			End:   0, // 00:00
		},
		{
			Start: 43200, // 12:00
			End:   0,     // 00:00
		},
	}

	for _, timeTable := range testData {
		result := timeTableInInterval(timeTable, testTime)

		if result != true {
			log.Fatalf("timeTableInInterval failed, start: %d, end: %d, current: %d", timeTable.Start, timeTable.End, getSecondsFromMidnight(testTime.Hour(), testTime.Minute()))
		}
	}
}

func TestTimeTableNotInInterval(t *testing.T) {
	testTime := time.Date(2023, 2, 4, 5, 30, 0, 0, time.UTC) // 05:30
	testData := [4]TimeTable{
		{
			Start: 36000, // 10:00
			End:   72000, // 20:00
		},
		{
			Start: 46800, // 13:00
			End:   10800, // 03:00
		},
		{
			Start: 25200, // 07:00
			End:   18000, // 05:00
		},
		{
			Start: 43200, // 12:00
			End:   0,     // 00:00
		},
	}

	for _, timeTable := range testData {
		result := timeTableInInterval(timeTable, testTime)

		if result != false {
			log.Fatalf("timeTableNotInInterval start: %d, end: %d, current: %d", timeTable.Start, timeTable.End, getSecondsFromMidnight(testTime.Hour(), testTime.Minute()))
		}
	}
}

func TestTemperatureUpdateNeeded(t *testing.T) {
	var update bool
	var temperature int
	testTimeTables := []TimeTable{
		{
			Start:       36000, // 10:00
			End:         72000, // 20:00
			Temperature: 25,
		},
	}
	testScheduler := TemperatureScheduler{
		Topic:              "myhome-kr/livingroom/danfoss-thermo-01",
		DefaultTemperature: 22,
		TimeTable:          testTimeTables,
	}

	// init current temperature state
	update, temperature = temperatureUpdateNeeded(testScheduler, time.Date(2023, 2, 4, 10, 59, 0, 0, time.UTC))
	if update {
		log.Fatal("temperatureUpdateNeeded #0 failed, update should be false")
	}

	// temperature state exists, should true because update is needed
	update, temperature = temperatureUpdateNeeded(testScheduler, time.Date(2023, 2, 4, 21, 00, 0, 0, time.UTC))
	if !update {
		log.Fatal("temperatureUpdateNeeded #1 failed, update should be true")
	} else if temperature != 22 {
		log.Fatalf("temperatureUpdateNeeded #1 failed, temperature should be 22, got: %d", temperature)
	}

	// one minutes later, should return false because no update is needed
	update, temperature = temperatureUpdateNeeded(testScheduler, time.Date(2023, 2, 4, 21, 01, 0, 0, time.UTC))
	if update {
		log.Fatal("temperatureUpdateNeeded #2 failed, update should be false")
	}

	// another day, update needed!
	update, temperature = temperatureUpdateNeeded(testScheduler, time.Date(2023, 2, 5, 10, 01, 0, 0, time.UTC))
	if !update {
		log.Fatal("temperatureUpdateNeeded #3 failed, update should be true")
	} else if temperature != 25 {
		log.Fatalf("temperatureUpdateNeeded #3 failed, temperature should be 25, got: %d", temperature)
	}
}

func TestChecksOverlaps(t *testing.T) {
	var t1, t2 TimeTable

	// should not overlap
	// t1 08:00 - 10:00
	// t2 11:00 - 12:00
	t1 = TimeTable{
		Start: 3600 * 8,
		End:   3600 * 10,
	}

	// should not overlap
	t2 = TimeTable{
		Start: 3600 * 11,
		End:   3600 * 12,
	}

	if checksOverlaps(t1, t2) {
		log.Fatal("checksOverlaps #0 failed, should not overlap")
	}

	// should overlap
	// t1 08:00 - 10:00
	// t2 08:30 - 11:00
	t1 = TimeTable{Start: 3600 * 8,
		End: 3600 * 10,
	}

	// should not overlap
	t2 = TimeTable{

		Start: 3600 * 8.5,
		End:   3600 * 11,
	}
	if !checksOverlaps(t1, t2) {
		log.Fatal("checksOverlaps #1 failed, should overlap")
	}

	// should overlap
	// t1 08:00 - 10:00
	// t2 06:00 - 08:30
	t1 = TimeTable{
		Start: 3600 * 8,
		End:   3600 * 10,
	}

	// should not overlap
	t2 = TimeTable{
		Start: 3600 * 6,
		End:   3600 * 8.5,
	}
	if !checksOverlaps(t1, t2) {
		log.Fatal("checksOverlaps #2 failed, should overlap")
	}

	// should overlap
	// t1 08:00 - 10:00
	// t2 06:00 - 12:00
	t1 = TimeTable{
		Start: 3600 * 8,
		End:   3600 * 10,
	}

	// should not overlap
	t2 = TimeTable{
		Start: 3600 * 6,
		End:   3600 * 12,
	}
	if !checksOverlaps(t1, t2) {
		log.Fatal("checksOverlaps #3 failed, should overlap")
	}
}
