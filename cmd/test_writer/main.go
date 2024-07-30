package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pircuser61/go_less/config"
)

func main() {

	ticker := time.NewTicker(time.Second * 5)
	defer func() {
		ticker.Stop()
		fmt.Println("writer done")
	}()

	f, err := os.OpenFile(config.GetFileIn(), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for {
		msg := fmt.Sprintln(time.Now().Format("2006-01-02 15:04:05"))
		if _, err = f.WriteString(msg); err != nil {
			panic(err)
		}
		select {
		case <-ticker.C:
			fmt.Println("TICK!", msg)
			continue
		}
	}
}
