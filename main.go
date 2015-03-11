package main

import (
	"./mqttc"
	"fmt"
	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	log "github.com/Sirupsen/logrus"
	"github.com/marpaia/graphite-golang"
	"gopkg.in/alecthomas/kingpin.v1"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	Graphite   *graphite.Graphite
	clientsMap map[*mqtt.MqttClient]*string
)

func sendMetrics(client *mqtt.MqttClient, msg mqtt.Message) {
	var host *string
	var ok bool
	if host, ok = clientsMap[client]; !ok {
		log.Warn("Not all the mqtt clients are ready")
		return
	}

	metric := normalizeMetric(msg.Topic())
	switch {
	case strings.Contains(metric, "version"):
		log.Debug("Skipping mosquitto version metric")
		return
	case strings.Contains(metric, "timestamp"):
		log.Debug("Skipping timestamp metric")
		return
	}

	value := string(msg.Payload())
	// mosquitto uptime metric appends 'seconds' to the number
	// we only need the value
	if metric == "mqtt.uptime" {
		value = strings.Split(string(msg.Payload()), " ")[0]
	}
	hostMetric := fmt.Sprintf("%s.%s", *host, metric)

	log.Debugf("Sending metric %s %s", hostMetric, value)
	err := Graphite.SimpleSend(hostMetric, value)
	if err != nil {
		log.Debugf("Error sending metric %s: %s", hostMetric, err)
	}
}

func parseBrokerUrls(brokerUrls string) []string {
	tokens := strings.Split(brokerUrls, ",")
	for i, url := range tokens {
		tokens[i] = strings.TrimSpace(url)
	}

	return tokens
}

func normalizeMetric(name string) string {
	m := strings.Replace(name, "$SYS/broker/", "mqtt.", -1)
	m = strings.Replace(m, "/", ".", -1)
	m = strings.Replace(m, " ", "_", -1)

	return m
}

func main() {
	kingpin.Version("0.1")

	brokerUrls := kingpin.Flag("broker-urls", "Comman separated MQTT broker URLs").
		Required().Default("").OverrideDefaultFromEnvar("MQTT_URLS").String()

	cafile := kingpin.Flag("cafile", "CA certificate when using TLS (optional)").
		String()

	graphiteHost := kingpin.Flag("graphiteHost", "Graphite host").
		Default("localhost").String()

	graphitePort := kingpin.Flag("graphitePort", "Graphite port").
		Default("2003").Int()

	graphitePing := kingpin.Flag("graphitePing", "Try to reconnect to graphite every X seconds").
		Default("15").Int()

	insecure := kingpin.Flag("insecure", "Don't verify the server's certificate chain and host name.").
		Default("false").Bool()

	debug := kingpin.Flag("debug", "Print debugging messages").
		Default("false").Bool()

	kingpin.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *cafile != "" {
		if _, err := os.Stat(*cafile); err != nil {
			log.Fatalf("Error reading CA certificate %s", err.Error())
			os.Exit(1)
		}
	}

	urlList := parseBrokerUrls(*brokerUrls)
	clientsMap = map[*mqtt.MqttClient]*string{}

	var gerr error
	Graphite, gerr = graphite.NewGraphite(*graphiteHost, *graphitePort)
	if gerr != nil {
		log.Warn("Error connecting to graphite")
		os.Exit(1)
	} else {
		log.Info("Connected to graphite")
		log.Debugf("Loaded Graphite connection: %#v", Graphite)
	}

	for _, urlStr := range urlList {
		args := mqttc.Args{
			BrokerURLs:    []string{urlStr},
			ClientID:      "mqtt-stats",
			Topic:         "$SYS/broker/#",
			TLSCACertPath: *cafile,
			TLSSkipVerify: *insecure,
		}

		uri, _ := url.Parse(urlStr)
		c := mqttc.Subscribe(sendMetrics, &args)
		defer c.Disconnect(0)
		host := strings.Split(uri.Host, ":")[0]
		host = strings.Replace(host, ".", "_", -1)
		clientsMap[c] = &host
	}

	// Try to reconnect every graphitePing sec if sending fails by sending
	// a fake ping metric
	// FIXME: better handled by the Graphite client
	go func() {
		for {
			time.Sleep(time.Duration(*graphitePing) * time.Second)
			err := Graphite.SimpleSend("ping metric", "")
			if err != nil {
				log.Warn("Ping metric failed, trying to reconnect")
				err = Graphite.Connect()
				if err != nil {
					log.Error("Reconnecting to graphite failed")
				} else {
					log.Info("Reconnected to graphite")
				}
			}
		}
	}()

	for {
		time.Sleep(1 * time.Second)
	}

}
