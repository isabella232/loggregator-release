package main

import (
	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/loggregator/counter/datadog"
	"code.cloudfoundry.org/rfc5424"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	c := config{
		TcpPort:         "8081",
		HttpPort:        "8080",
		CounterInterval: 1 * time.Minute,
	}

	err := envstruct.Load(&c)
	if err != nil {
		panic(err)
	}

	counts := &counts{
		tags: strings.Split(c.Tags, ","),
	}

	go startTCPServer(c.TcpPort, counts)
	go startHTTPServer(c.HttpPort, counts)

	// TODO
	// streams logs - RLP client
	// need sourceID of the app

	datadogReporter := datadog.NewReporter(c.DatadogApiKey, counts, datadog.WithInterval(c.CounterInterval))
	datadogReporter.Run()
}

type config struct {
	TcpPort         string        `env:"TCP_PORT"`
	HttpPort        string        `env:"HTTP_PORT"`
	Tags            string        `env:"TAGS"`
	DatadogApiKey   string        `env:"DATADOG_API_KEY"`
	CounterInterval time.Duration `env:"COUNTER_INTERVAL"`
}

type counts struct {
	httpCounter uint64
	tcpCounter  uint64
	tags        []string
}

func (c *counts) BuildPoints(timestamp int64) ([]datadog.Point, error) {
	return []datadog.Point{
		{
			Metric: "syslog_blackbox.tcp_count",
			Points: [][]int64{{timestamp, c.GetTCP()}},
			Type:   "counter",
			Tags:   c.tags,
		},
		{
			Metric: "syslog_blackbox.http_count",
			Points: [][]int64{{timestamp, c.GetHTTP()}},
			Type:   "counter",
			Tags:   c.tags,
		},
	}, nil
}

func (c *counts) IncTCP() {
	atomic.AddUint64(&c.tcpCounter, 1)
}

func (c *counts) IncHTTP() {
	atomic.AddUint64(&c.httpCounter, 1)
}

func (c *counts) GetTCP() int64 {
	return int64(atomic.LoadUint64(&c.tcpCounter))
}

func (c *counts) GetHTTP() int64 {
	return int64(atomic.LoadUint64(&c.httpCounter))
}

func startTCPServer(port string, counts *counts) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleTCPConnection(conn, counts)
	}
}

func startHTTPServer(port string, counts *counts) {

	countHandler := func(w http.ResponseWriter, req *http.Request) {
		counts.IncHTTP()
	}

	http.HandleFunc("/", countHandler)
	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		panic(err)
	}
}

func handleTCPConnection(conn net.Conn, counts *counts) {
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer conn.Close()

	var msg rfc5424.Message
	for {
		_, err := msg.ReadFrom(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("ReadFrom err: %s", err)
			return
		}

		counts.IncTCP()
	}
}
