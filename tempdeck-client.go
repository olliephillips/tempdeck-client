// tempdeck-client. An MQTT monitoring application for tempdeck
// Sets up an MQTT client and webserver and passes MQTT messages to the browser session over websocket
// Ollie Phillips 2016, MIT license
package main

import (
	"log"
	"fmt"
	"os"
	"os/signal"
	"net/http"
	"html/template"
	"github.com/spf13/cobra"
	"github.com/yosssi/gmq/mqtt"
    "github.com/yosssi/gmq/mqtt/client"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

// For cobra command line switches
var (
	broker string
	serial string
)

// Messages channel
var messages = make(chan string)

// Home handler
func home(w http.ResponseWriter, req *http.Request){
	// Quick.Dirty. Don't want external dependencies for the server.
	const html = `
<!DOCTYPE html>
<html>
<head>
<title>Tempdeck-client: Monitoring {{.Topic}}{{.Serial}}</title>
<script src="http://canvasjs.com/assets/script/canvasjs.min.js"></script>
<link href='https://fonts.googleapis.com/css?family=Roboto+Slab:400,300,100,700' rel='stylesheet' type='text/css'>
<style>
body{background-color:#B02731;color:#F9FFC3}
h1{font-size:40px; color:#FC3846;}
h1,h2, h3, p {font-family: "Roboto Slab", serif;}
#header{width:50%;margin:0px auto 0px auto; text-align:center;}
#body{padding-top:20px;width:100%;}
#chartarea{height:400px;width:70%;margin:0px auto 0px auto;}
#footer{position:absolute; bottom: 0px; left:0px; height:75px;width:100%;background-color:#70191F;}
#footerContent{padding-top:20px;text-align:center;}
.title{color:#FFA498;}
</style>
</head>
<body>
<div id="header">
	<h1>Tempdeck</h1>
	<h3><span class="title">Broker:</span> {{.Broker}}<br/>
	<span class="title">Topic:</span> {{.Topic}}{{.Serial}}</h3>
</div>	
<div id="body">
	<div id="chartarea">
	</div>
</div>
<div id="footer">
	<div id="footerContent">
		<p>http://github.com/olliephillips/tempdeck</p>
	</div>
</div>
<script>
window.onload = function () {
	
	// Data arrays
	var dataPoints1 = [];
	var dataPoints2 = [];
	
	// Configure chart
	var chart = new CanvasJS.Chart("chartarea",{
			backgroundColor: "#FC3846",
			zoomEnabled: true,
			title: {
				text: "Temperature"	,
				fontSize: 22,	
				fontFamily: "Roboto Slab",
				fontColor: "white"
			},
			toolTip: {
				enabled: false,
				shared: false
			},
			legend: {
				fontFamily: "Roboto Slab",
				fontColor: "white",
				verticalAlign: "bottom",
				horizontalAlign: "center",
 				fontSize: 14
			},
			axisY:{
				includeZero: false,
				suffix: "C",
				labelFontColor: "white",
				labelFontFamily: "Roboto Slab",
				labelFontSize: 16,
				gridColor: "#FF5047"
			}, 
			axisX:{
				labelFontColor: "white",
				labelFontFamily: "Roboto Slab",
				labelFontSize: 16,
				margin:10
			}, 
			data: [{ 
				// dataSeries1
				color: "#F9FFC3",
				type: "stepLine",
				xValueType: "dateTime",
				showInLegend: true,
				name: "Actual",
				dataPoints: dataPoints1	,	
			},
			{				
				// dataSeries2
				color: "#FFA498", 
				type: "stepLine",
				xValueType: "dateTime",
				showInLegend: true,
				name: "Target" ,
				dataPoints: dataPoints2			
			}]
	});

	// Update routine called on websocket message	
	var updateChart = function (json) {
		var data = JSON.parse(json);
		var time = new Date;
		var dpTime = time.getTime();
		
		dataPoints1.push({
 			x: dpTime,
 			y: parseFloat(data["currentTemp"])
 		});
 		dataPoints2.push({
 			x: dpTime,
 			y: parseFloat(data["targetTemp"])
 		});
		
		// Rerender	
 		chart.render();
	}	
	
	// Ensure initial render
	chart.render();

	// Websocket
	var ws = {
		conn: null,
		connect: function(){
			
			//var username = 'tempdeck-client';
			var socket = 'ws://localhost:8081/ws';
		
			ws.conn = new WebSocket(socket);
		
			// Log errors
			ws.conn.onerror = function (error) {
		  		console.log('WebSocket Error ' + error);
			};

			// Handle messages from the server
			ws.conn.onmessage = function (e) {
				//console.log(e.data);
				updateChart(e.data);
			};
		}
	};
	
	// Initialise websocket
	ws.connect();
}
</script>
</body>
</html>
	`
	t, err := template.New("homepage").Parse(html)
	
	data := struct {
		Topic string
		Serial string
		Broker string
	}{
		Topic : "tempdeck/espruino/",
		Serial: serial,
		Broker: broker,
	}
	
	err = t.Execute(w, data)
	checkError(err)
}

// Handler/Upgrader/Watcher for websocket connections
func wsHandler(w http.ResponseWriter, req *http.Request){
	conn, _ := upgrader.Upgrade(w, req, nil)
	for {
		//Read from channel
		message:= <-messages
   		conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}

// Create webserver 
func startHTTPServer(){
	// Define two routes
	http.HandleFunc("/ws", wsHandler) 
	http.HandleFunc("/", home)
	log.Printf("Starting tempdeck-client app on http://localhost:8081")
	go http.ListenAndServe("localhost:8081", nil)
}

// Setup and start MQTT client, put received messages on the channel
func startMQTT(){
	// Set up channel on which to send signal notifications
    sigc := make(chan os.Signal, 1)
    signal.Notify(sigc, os.Interrupt, os.Kill)

    cli := client.New(&client.Options{
        // Define the processing of the error handler
        ErrorHandler: func(err error) {
            fmt.Println(err)
        },
    })	
    defer cli.Terminate()

    // Connect
    err := cli.Connect(&client.ConnectOptions{
        Network:  "tcp",
        Address:  broker + ":1883",
        ClientID: []byte("example-client"),
    })
   	checkError(err)
	
    // Subscribe to our topic
    err = cli.Subscribe(&client.SubscribeOptions{
        SubReqs: []*client.SubReq{
            &client.SubReq{
                TopicFilter: []byte("tempdeck/espruino/" + serial),
                QoS:         mqtt.QoS0,
                // Message handler
                Handler: func(topicName, message []byte) {
					// Put the message in the "messages" channel	
					messages <- string(message) 
                },
            },
        },
    })
    checkError(err)
	
	// Wait for receiving a signal
    <-sigc

    // Disconnect the Network Connection
    if err := cli.Disconnect(); err != nil {
        panic(err)
    }
}

// Utility function for error checking
func checkError(err error){
	if err != nil {
		log.Fatal(err)
	}
}	

// Start, uses Cobra to handle command line switches
func main() {
	cmd := &cobra.Command{
				Use:   "tempdeck-client",
				Short: "tempdeck-client is an MQTT client built specifically for monitoring Espruino powered devices running tempdeck",
				Run: func(cmd *cobra.Command, args []string) {		
					// Start web server
					go startHTTPServer() 
					
					// Create MQTT client
					startMQTT()
				},
			}
	cmd.Flags().StringVarP(&broker, "broker", "b", "test.mosquitto.org", "MQTT broker")
	cmd.Flags().StringVarP(&serial, "serial", "s", "18fe34da-fa4a", "Serial number of your Espruino board")
	cmd.Execute()
}	