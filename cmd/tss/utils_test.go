package main

import (
	"reflect"
	"testing"
)

func TestParseSyncConfig(t *testing.T) {
	type args struct {
		jsonStr string
	}
	tests := []struct {
		name    string
		args    args
		want    SensorTrvSync
		wantErr bool
	}{
		{name: "Parse config", args: args{jsonStr: `{ "sensor-topic": "topic1", "trv-topic": "topic2" }`}, want: SensorTrvSync{SensorTopic: "topic1", TrvTopic: "topic2"}, wantErr: false},
		{name: "Parse config - err no sensor topic", args: args{jsonStr: `{ "sensor-topic": "", "trv-topic": "topic2" }`}, want: SensorTrvSync{}, wantErr: true},
		{name: "Parse config - err no trv topic", args: args{jsonStr: `{ "sensor-topic": "topic1", "trv-topic": "" }`}, want: SensorTrvSync{}, wantErr: true},
		{name: "Parse config - err invalid json", args: args{jsonStr: `{ "sensor-topic": "topic1", `}, want: SensorTrvSync{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSyncConfig(tt.args.jsonStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSyncConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSyncConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
