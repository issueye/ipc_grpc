package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/issueye/ipc_grpc/client"
	"github.com/issueye/ipc_grpc/server"
)

func main() {

	var (
		srv *server.Server
		err error
	)

	srv, err = server.NewServer(true, &server.Host{}, func(pi server.PluginInfo) error {
		// return errors.New("检查不通过")
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

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

	err = c.Register()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = c.Heartbeat()
		if err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(20 * time.Second)

	data, err := json.Marshal(srv.Plugins)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}
