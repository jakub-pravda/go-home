# Home-Aut services

Set of simple microservices for home automation

## TSS (TRV, Sensor synchronizer)

Service sends data from an external room sensor to a thermostatic radiator valve (TRV). TRV is often located near a radiator and
 its temperature measuring is affected by this. Another room temperature sensor located far from a heat source is then recommended.

```bash
go run cmd/tss/main.go \
--broker 'tcp://localhost:1883' \
--sensor-topic 'myhome-kr/livingroom/son-sns-01' \
--trv-topic 'myhome-kr/livingroom/danfoss-thermo-01' \
--cron '*/15 * * * *'
```
