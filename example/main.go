package main

import (
	"fmt"
	"log"
	"time"

	"github.com/issueye/ipc_grpc/client"
	"github.com/issueye/ipc_grpc/server"
)

func main() {

	go server.NewServer(&server.Host{})

	c, err := client.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	c.SetHeartbeatInterval(3 * time.Second)

	pingResp, err := c.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(pingResp.Message, pingResp.Timestamp)

	err = c.Heartbeat()
	if err != nil {
		log.Fatal(err)
	}

	err = c.SendInfo()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(30 * time.Second)
}
