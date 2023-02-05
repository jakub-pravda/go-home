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
	schedulers schedulersConfigs

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
	// TODO check if the same topic is not already in the list
	// TODO check overlapping time ranges
	*i = append(*i, result)
	return nil
}

func checkAndUpdate(client MQTT.Client, schedulers schedulersConfigs) {
	for _, scheduler := range schedulers {
		update, temperature := temperatureUpdateNeeded(scheduler, time.Now())
		if update {
			heatingSetpointTopic := fmt.Sprintf("%s/set/occupied_heating_setpoint_scheduled", scheduler.Topic)
			log.Printf("Updating %s to %d°C", heatingSetpointTopic, temperature)
			client.Publish(heatingSetpointTopic, 0, false, temperature)
		}
	}
}

func main() {
	log.Printf("=== Starting TRV temperature scheduler ===")

	flag.Var(&schedulers, "scheduler", "Scheduler configuration (use format json formatted string: '{\"topic\": \"topic1\", \"defaultTemperature\": 22, \"timeTable\": [{\"start\": \"22:30\", \"end\": \"05:30\", \"temperature\": 18}]}'))")
	flag.Parse()

	log.Printf("Schedulers: %v", schedulers)
	log.Printf("MQTT broker host: %s", *mqttBroker)

	connOpts := MQTT.NewClientOptions().AddBroker(*mqttBroker).SetClientID("tsc").SetCleanSession(true)
	client := MQTT.NewClient(connOpts)

	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(1).Minute().Do(checkAndUpdate, client, schedulers)

	wait := utils.GracefulShutdown(context.Background(), 30*time.Second, map[string]utils.Operation{
		"close-mqtt": func(ctx context.Context) error {
			client.Disconnect(0)
			return nil
		},
	})
	<-wait
}
