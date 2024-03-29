# Home-Aut services

Set of simple microservices for home automation

## TSS (TRV, Sensor synchronizer)

Service sends data from an external room sensor to a thermostatic radiator valve (TRV). TRV is often located near a radiator and
 its temperature measuring is affected by this. Another room temperature sensor located far from a heat source is then recommended.

```bash
go run cmd/tss/main.go \
--broker 'tcp://localhost:1883' \
--cron '*/15 * * * *' \
--sync '{ "sensor-topic": "myhome-kr/livingroom/son-sns-01", "trv-topic": "myhome-kr/livingroom/danfoss-thermo-01" }' \
--sync '{ "sensor-topic": "myhome-kr/livingroom/son-sns-02", "trv-topic": "myhome-kr/livingroom/danfoss-thermo-02" }' 
```

## TSC (Temperature scheduler)

Service schedules temperature changes for a TRV. It's possible to set a default temperature and a time table with temperature changes.

```bash
 go run ./cmd/tsc/main.go ./cmd/tsc/utils.go --scheduler '{ "topic": "myhome-kr/livingroom/danfoss-thermo-01", "defaultTemperature": 22, "timeTable": [ { "start": "22:00", "end": "06:00", "temperature": 18 } ] }'
```

## Nix

It's possible to build nix derivation by following set of commands

```
$ nix develop
$ gomod2nix
```
