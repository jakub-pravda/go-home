package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jacfal.io/homeaut/pkg/sensors"
	"github.com/jacfal.io/homeaut/utils"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const termSensorTimeoutSeconds = 60 * 60 * 3 // 3 hours 
const QOS = 0

var (
	currentTemperature float32 = sensors.ExternalSensorUndefined
	termSensorUpdateUnix int64 = 0

	mu sync.Mutex

	// input args
	mqttBroker 	= flag.String("broker", "tcp://localhost:1883", "MQTT broker connection string")
	sensorTopic = flag.String("sensor-topic", "", "MQTT sensor topic to subscribe") 
	trvTopic	= flag.String("trv-topic", "", "Thermostatic radiator valve topic to publish")
	cron		= flag.String("cron", "", "Interval of sending sensor temp data to the TRV (use cron format '*/15 * * * *')")
)

func onSensorMessageReceived(client MQTT.Client, message MQTT.Message) {
	setTempVar := func(temp float32) {
		mu.Lock()
		defer mu.Unlock()

		log.Printf("Setting new current temp = %f°C", temp)
		currentTemperature = temp
		termSensorUpdateUnix = time.Now().Unix()
	}
	
	log.Printf("Sensor message received: %s", message.Payload())
	sonoffPayload, err := sensors.SonoffSensorPayloadToStruct(string(message.Payload()))
	if err != nil {
		log.Println("Error! Can't parse sensor payload")
	} else {
		setTempVar(sonoffPayload.Temperature)
	}
}

func sensorTempTRV(topic string, client MQTT.Client) func() {
	termSensorTimeoutSeconds := 60 * 60 * 3 // 3 hours

	// closure
	return func() {
		mu.Lock()
		defer mu.Unlock()

		termSensorUpdateDeltaUnix := time.Now().Unix() - termSensorUpdateUnix
		if currentTemperature != sensors.ExternalSensorUndefined && termSensorUpdateDeltaUnix > int64(termSensorTimeoutSeconds) {
			log.Printf("Warning! Haven't received data from sensor for %d minutes. Disassembling tandem", (termSensorUpdateDeltaUnix / 60))
			currentTemperature = sensors.ExternalSensorUndefined
		}

		log.Printf("Sending current sensor temp (%.2f°C) to the thermo head", currentTemperature)
		if token := client.Publish(topic, QOS, false, fmt.Sprintf("%d", sensors.GetExternalTempSensorFormat(currentTemperature))); token.Wait() && token.Error() != nil {
			log.Printf("Error! Publish sensor temperature failed. Topic %s, temperature: %f", topic, currentTemperature)
		}
	}
}

// TODO Open telemetry SDK

func main() {
	log.Printf("=== Starting Thermo head <---> Sensor synchronizer ===")
	flag.Parse()

	log.Printf("MQTT broker host: %s", *mqttBroker)

	if *sensorTopic == "" {
		log.Fatalf("Error! Sensor topic must be set")
	} else {
		log.Printf("Sensor topic to subscribe: %s", *sensorTopic)
	}

	if *trvTopic == "" {
		log.Fatalf("Error! Thermostatic radiator valve topic must be set")
	} else {
		log.Printf("Thermostatic radiator valve topic to subscribe: %s", *sensorTopic)
	}

	if *cron == "" {
		log.Fatalf("Error! Sensor --> TRV sync interval must be set")
	} else {
		log.Printf("Sensor --> TRV sync interval: %s", *cron)
	}

	scheduler := gocron.NewScheduler(time.UTC)
	externalMeasuredRoomSensorTopic := fmt.Sprintf("%s/set/external_measured_room_sensor", *sensorTopic)

	connOpts := MQTT.NewClientOptions().AddBroker(*mqttBroker).SetClientID("test-client").SetCleanSession(true).SetDefaultPublishHandler(onSensorMessageReceived)

	// MQTT Broker - topic subscription settings
	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(*sensorTopic, QOS, onSensorMessageReceived); token.Wait() && token.Error() != nil {
			log.Panicf("Error, topic %s subscription failed: %s", *sensorTopic ,token.Error())
		} else {
			log.Printf("Topic %s subscribed", *sensorTopic)
		}
	}

	// MQTT Broker - connect to the client, subscribe topic
	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error, broker connection failed: %s", token.Error())
	} else {
		log.Printf("Connected to the MQTT broker")
	}

	scheduler.Cron(*cron).Do(sensorTempTRV(externalMeasuredRoomSensorTopic, client))
	scheduler.StartAsync()

	// wait for termination signal and register database & http server clean-up operations
	wait := utils.GracefulShutdown(context.Background(), 30 * time.Second, map[string]utils.Operation{
	"disassemble-and-close": func(ctx context.Context) error {
		defer client.Disconnect(0)
		token := client.Publish(externalMeasuredRoomSensorTopic, 0, false, fmt.Sprintf("%d", sensors.ExternalSensorUndefined))
		token.Wait()
		return token.Error()
		},
	})
	
	<-wait
}
