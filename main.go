package main

import (
	"log"
)

func main() {
	config := LoadConfig()
	service := NewConservationService(NewArtifactStore())
	app := newOpsApp()
	app.Worker.Start()
	app.Heartbeat.Start()
	log.Printf("museum conservation service listening on :%s", config.Port)
	err := serveAddress(":"+config.Port, NewRouter(service, app))
	app.Worker.Stop()
	app.Heartbeat.Stop()
	if err != nil {
		log.Fatal(err)
	}
}
