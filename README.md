# tempdeck-client

An MQTT monitoring application for tempdeck (https://github.com/olliephillips/tempdeck)

tempdeck-client provides an MQTT client and webserver and passes MQTT messages to the browser session over websocket. If you are running tempdeck this application will allow you to watch and monitor the realtime updates.

## Setup
This is a Go application, you'll need a Go enviroment as there are no binaries currently

```
go get github.com/olliephillips/tempdeck-client
```
Run it or build and install a binary in the usual way.

## Usage

```
tempdeck-client -h
```

```
Usage:
  tempdeck-client [flags]

Flags:
  -b, --broker="test.mosquitto.org": MQTT broker
  -s, --serial="18fe343-fa4a": Serial number of your Espruino board
```

