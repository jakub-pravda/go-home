package sensors

import (
	"encoding/json"
	"errors"
)

type SonoffTemperatureSensor struct {
	Battery     float32 `json:"battery"`
	Humidity    float32 `json:"humidity"`
	Linkquality int     `json:"linkquality"`
	Temperature float32 `json:"temperature"`
	Voltage     int     `json:"voltage"`
}

// ISensor interface
func getTemperature(s *SonoffTemperatureSensor) float32 {
	return s.Temperature
}

func SonoffSensorPayloadToStruct(mqttPayload string) (SonoffTemperatureSensor, error) {
	var sns SonoffTemperatureSensor
	err := json.Unmarshal([]byte(mqttPayload), &sns)
	if err != nil {
		return SonoffTemperatureSensor{}, errors.New("Invalid payload. Not a sonoff temperature sensor format")
	}
	return sns, nil
}
