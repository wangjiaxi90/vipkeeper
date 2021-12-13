package main

import (
	"fmt"
	"github.com/wangjiaxi90/vipkeeper/vip-keeper"
	"log"
	"os"
)

const (
	version = "0.0.2"
)

func main() {
	if (len(os.Args) > 1) && (os.Args[1] == "--version") {
		fmt.Printf("version: %s\n", version)
		return
	}
	var err error
	conf, err := vip_keeper.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	v, err := vip_keeper.NewVipKeeper(conf)
	if err != nil {
		log.Fatal(err)
	}
	v.Start()
}
