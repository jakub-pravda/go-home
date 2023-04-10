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

// Tandem definition Sensor --> TRV
type SensorTrvSync struct {
	SensorTopic string `json:"sensor-topic"`
	TrvTopic    string `json:"trv-topic"`
}

// SensorTemperature holds current temperature and last update time
type SensorTemperature struct {
	temperature    float32
	lastUpdateUnix int64
}

type SyncConfigs []SensorTrvSync

var (
	// key: trv topic, value: sensor temperature
	sensorTemperatures = map[string]SensorTemperature{}
	sensorTrvTopic     = map[string]string{}
	mu                 sync.Mutex

	// input args
	mqttBroker = flag.String("broker", "tcp://localhost:1883", "MQTT broker connection string")
	cron       = flag.String("cron", "", "Interval of sending sensor temp data to the TRV (use cron format '*/15 * * * *')")
	syncs      SyncConfigs
)

func sensorTempTRV(client MQTT.Client) func() {
	// Data receive timeout, when data not received from sensor within this time, we disassemble tandem
	termSensorTimeoutSeconds := 60 * 60 * 3 // 3 hours

	// closure
	return func() {
		mu.Lock()
		defer mu.Unlock()

		for sensorTopic, sensorData := range sensorTemperatures {
			termSensorUpdateDeltaUnix := time.Now().Unix() - sensorData.lastUpdateUnix
			if sensorData.temperature != sensors.ExternalSensorUndefined && termSensorUpdateDeltaUnix > int64(termSensorTimeoutSeconds) {
				log.Printf("Warning! Haven't received data from sensor for %d minutes. Disassembling tandem", (termSensorUpdateDeltaUnix / 60))
				// we need to tell TRV that sensor isn't available
				sensorData = SensorTemperature{temperature: sensors.ExternalSensorUndefined, lastUpdateUnix: 0}
			}

			trvTopic := sensorTrvTopic[sensorTopic]
			if trvTopic == "" {
				log.Printf("Error! Can't find TRV topic for sensor %s", sensorTopic)
				continue
			}

			log.Printf("Sending current sensor temp (%.2f°C) to the thermo head (%s)", sensorData.temperature, trvTopic)
			if token := client.Publish(trvTopic, QOS, false, fmt.Sprintf("%d", sensors.GetExternalTempSensorFormat(sensorData.temperature))); token.Wait() && token.Error() != nil {
				log.Printf("Error! Publish sensor temperature failed. Topic %s, temperature: %f", trvTopic, sensorData.temperature)
			}
		}
	}
}

func (i *SyncConfigs) String() string {
	// not used, but required by flag.Var
	return ""
}

func (i *SyncConfigs) Set(value string) error {
	result, err := parseSyncConfig(value)
	if err != nil {
		log.Printf("Can't parse sync configs: %v", err)
		return err
	}
	*i = append(*i, result)
	return nil
}

func main() {
	log.Printf("=== Starting Thermo head <---> Sensor synchronizer ===")

	flag.Var(&syncs, "sync", "Sensor, TRV sync json config: { 'sensor-topic': 'myhome-kr/livingroom/son-sns-01', 'trv-topic': 'myhome-kr/livingroom/danfoss-thermo-01' }")
	flag.Parse()

	log.Printf("Sync configs: %v", syncs)
	log.Printf("MQTT broker host: %s", *mqttBroker)

	if *cron == "" {
		log.Fatalf("Error! Sensor --> TRV sync interval must be set")
	} else {
		log.Printf("Sensor --> TRV sync interval: %s", *cron)
	}

	scheduler := gocron.NewScheduler(time.UTC)
	connOpts := MQTT.NewClientOptions().AddBroker(*mqttBroker).SetClientID("test-client").SetCleanSession(true)

	// MQTT Broker - topic subscription settings
	connOpts.OnConnect = func(c MQTT.Client) {

		topicsToSubscribe := map[string]byte{}
		for _, syncConfig := range syncs {
			topicsToSubscribe[syncConfig.SensorTopic] = QOS
			sensorTrvTopic[syncConfig.SensorTopic] = fmt.Sprintf("%s/set/external_measured_room_sensor", syncConfig.TrvTopic)
		}

		log.Printf("Paired topics %v", sensorTrvTopic)
		var onSensorMessageReceived = func(client MQTT.Client, message MQTT.Message) {
			setTempVar := func(temp float32) {
				mu.Lock()
				defer mu.Unlock()
				log.Printf("Setting new current temp = %f°C (%s)", temp, message.Topic())
				sensorTemperatures[message.Topic()] = SensorTemperature{temp, time.Now().Unix()}
			}
			log.Printf("Sensor message received: %s (%s)", message.Payload(), message.Topic())
			sonoffPayload, err := sensors.SonoffSensorPayloadToStruct(string(message.Payload()))
			if err != nil {
				log.Printf("Error! Can't parse sensor payload (%s)", message.Topic())
			} else {
				setTempVar(sonoffPayload.Temperature)
			}
		}

		if token := c.SubscribeMultiple(topicsToSubscribe, onSensorMessageReceived); token.Wait() && token.Error() != nil {
			log.Panicf("Error, topics %v subscription failed: %s", topicsToSubscribe, token.Error())
		} else {
			log.Printf("Topic %v subscribed", topicsToSubscribe)
		}

	}

	// MQTT Broker - connect to the client, subscribe topic
	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error, broker connection failed: %s", token.Error())
	} else {
		log.Printf("Connected to the MQTT broker")
	}

	scheduler.Cron(*cron).Do(sensorTempTRV(client))
	scheduler.StartAsync()

	// wait for termination signal and register database & http server clean-up operations
	wait := utils.GracefulShutdown(context.Background(), 30*time.Second, map[string]utils.Operation{
		"disassemble-and-close": func(ctx context.Context) error {
			defer client.Disconnect(0)
			for trvTopic, _ := range sensorTemperatures {
				// disassemble all sensor --> TRV tandems
				token := client.Publish(trvTopic, 0, false, fmt.Sprintf("%d", sensors.ExternalSensorUndefined))
				token.Wait()
				if token.Error() != nil {
					return token.Error()
				}
			}
			return nil
		},
	})

	<-wait
}
