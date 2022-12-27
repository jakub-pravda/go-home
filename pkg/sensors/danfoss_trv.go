package sensors

const ExternalSensorUndefined = -8000

func GetExternalTempSensorFormat(temperature float32) int {
	if temperature == ExternalSensorUndefined {
		return ExternalSensorUndefined
	} else {
		toReturn := int(temperature * 100)
		if toReturn < ExternalSensorUndefined || toReturn > 3500 {
			return ExternalSensorUndefined
		}
		return toReturn
	}
}
