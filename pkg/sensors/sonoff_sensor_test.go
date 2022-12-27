package sensors

import (
	"log"
	"testing"
)

func TestIfPayloadParsedCorrectly(t *testing.T) {
	testPayload := "{\"battery\":72.5,\"humidity\":64.75,\"linkquality\":108,\"temperature\":21.44,\"voltage\":2900}"
	
	// success
	expected := SonoffTemperatureSensor{
		Battery:     72.5,
		Humidity:    64.75,
		Linkquality: 108,
		Temperature: 21.44,
		Voltage:     2900,
	}

	result, err := SonoffSensorPayloadToStruct(testPayload)

	if err != nil || result != expected {
		log.Fatalf("Sonoff payload parsing failed- should be fine")
	}
	
	// failed
	testPayload = "{ just some invalid text }"
	_, err = SonoffSensorPayloadToStruct(testPayload)
	if err == nil {
		log.Fatalf("Sonoff payload parsing failed - should be err")
	}
}
