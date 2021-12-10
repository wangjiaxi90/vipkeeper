package main

import (
	"fmt"
	"github.com/wangjiaxi90/vipkeeper/vipkeeper"
	"log"
	"os"
)

const (
	version = "0.0.1"
)

func main() {
	if (len(os.Args) > 1) && (os.Args[1] == "--version") {
		fmt.Printf("version: %s\n", version)
		return
	}
	var err error
	conf, err := vipkeeper.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	v, err := vipkeeper.NewVipKeeper(conf)
	if err != nil {
		log.Fatal(err)
	}
	v.Start()
}
