package main

import (
	"context"
	loc "github.com/tinygoprogs/sigint/wifi/location"
	"time"
)

func main() {
	c := loc.Config{}
	c.UpdateInterval = time.Second * 1
	p := loc.NewProvider(c)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer func() {
		print("done")
		cancel()
	}()
	go p.Run(ctx)
	t := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-ctx.Done():
			println("background + 20 seconds done")
			return
		case <-t.C:
			println("Location:", p.RetrieveLocation())
		}
	}
}
