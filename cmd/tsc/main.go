package main

// TODO not sure if this is ok (to have multiple mains in one project)

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jacfal.io/homeaut/utils"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var (
	temperatureSchedulers schedulersConfigs

	// input args
	mqttBroker = flag.String("broker", "tcp://localhost:1883", "MQTT broker connection string")
)

type schedulersConfigs []TemperatureScheduler

func (i *schedulersConfigs) String() string {
	return "my string representation"
}

func (i *schedulersConfigs) Set(value string) error {
	result, err := parseTimeTable(value)
	if err != nil {
		log.Fatalf("Can't parse scheduler config: %v", err)
	}
	*i = append(*i, result)
	return nil
}

func checkAndUpdate(client MQTT.Client, schedulers schedulersConfigs) {
	for _, scheduler := range schedulers {
		update, temperature := temperatureUpdateNeeded(scheduler, time.Now())
		if update {
			heatingSetpointTopic := fmt.Sprintf("%s/set/occupied_heating_setpoint_scheduled", scheduler.Topic)
			log.Printf("Updating %s to %d°C", heatingSetpointTopic, temperature)
			if token := client.Publish(heatingSetpointTopic, 0, false, fmt.Sprintf("%d", temperature)); token.Wait() && token.Error() != nil {
				log.Printf("Error publishing to topic %s: %v", heatingSetpointTopic, token.Error())
			} else {
				log.Printf("Published temperature %d°C to topic %s", temperature, heatingSetpointTopic)
			}
		}
	}
}

func main() {
	log.Printf("=== Starting TRV temperature scheduler ===")

	flag.Var(&temperatureSchedulers, "scheduler", "Scheduler configuration (use format json formatted string: '{\"topic\": \"topic1\", \"defaultTemperature\": 22, \"timeTable\": [{\"start\": \"22:30\", \"end\": \"05:30\", \"temperature\": 18}]}'))")
	flag.Parse()

	log.Printf("Schedulers: %v", temperatureSchedulers)
	log.Printf("MQTT broker host: %s", *mqttBroker)

	// check schedulers overlaps
	for _, temperatureScheduler := range temperatureSchedulers {
		checkTimeTableOverlap(temperatureScheduler)
	}

	connOpts := MQTT.NewClientOptions().AddBroker(*mqttBroker).SetClientID("tsc").SetCleanSession(true)
	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error, broker connection failed: %s", token.Error())
	} else {
		log.Printf("Connected to the MQTT broker")
	}

	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(1).Minute().Do(checkAndUpdate, client, temperatureSchedulers)
	scheduler.StartAsync()

	wait := utils.GracefulShutdown(context.Background(), 30*time.Second, map[string]utils.Operation{
		"close-mqtt": func(ctx context.Context) error {
			client.Disconnect(0)
			return nil
		},
	})
	<-wait
}
