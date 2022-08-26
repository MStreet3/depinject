package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mstreet3/depinject/config"
	worker "github.com/mstreet3/depinject/worker/v6"
)

func main() {
	var (
		ctxwt, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		cfg           = config.NewConfig()
		ris, err      = worker.NewRandIntStream(cfg)
		values        = ris.Start(ctxwt)
	)

	defer cancel()

	if err != nil {
		log.Fatal(err)
	}

	for val := range values {
		fmt.Println(val)
	}
}
