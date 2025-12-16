package main

import (
	"log"

	"zte-c320-snmp-api/internal/api"
	"zte-c320-snmp-api/internal/cfg"
)

func main() {
	loader, err := cfg.NewLoader("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := api.NewRouter(loader)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
