package vip_keeper

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

type VipKeeper struct {
	conf     *Config
	client   *clientv3.Client
	ctx      context.Context
	cancel   context.CancelFunc
	vip      net.IP
	vipMask  net.IPMask
	netIface *net.Interface
}

func NewVipKeeper(conf *Config) (*VipKeeper, error) {
	vip := net.ParseIP(conf.IP)
	vipMask := getMask(vip, conf.Mask)
	netIface := getNetIface(conf.Iface)
	var err error
	var cli *clientv3.Client

	if conf.User != "" {
		if conf.Password != "" {
			cli, err = clientv3.New(clientv3.Config{Endpoints: conf.Endpoints, Username: conf.User, Password: conf.Password})
		} else {
			err = fmt.Errorf("Conf has etcd username but don't have passowrd. ")
		}
	} else {
		cli, err = clientv3.New(clientv3.Config{Endpoints: conf.Endpoints})
	}

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	v := &VipKeeper{
		conf:     conf,
		client:   cli,
		ctx:      ctx,
		cancel:   cancel,
		vip:      vip,
		vipMask:  vipMask,
		netIface: netIface,
	}

	return v, nil
}

func (v *VipKeeper) Start() {
	defer func(client *clientv3.Client) {
		err := client.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(v.client)
	v.campaign()
	v.receiveKillSignal()

}

func (v *VipKeeper) receiveKillSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Println("Received exit signal")
	v.cancel()
}

func (v *VipKeeper) campaign() {
	for {
		s, err := concurrency.NewSession(v.client, concurrency.WithTTL(v.conf.Interval/1000*15)) //TODO TTL is not a variable
		if err != nil {
			fmt.Println(err)
			continue
		}
		e := concurrency.NewElection(s, prefix)

		if err = e.Campaign(v.ctx, prop); err != nil {
			fmt.Println(err)
			continue
		}
		log.Println("elect: success")
		if success := v.runAddressConfiguration("add"); !success {
			log.Println("Add vip failed. Program will exit with cancel context. Please check your network state.")
			v.cancel()
		}

		select {
		case <-s.Done(): // 如果因为网络因素导致与etcd断开了keepAlive，这里break，重新创建session，重新选举
			log.Println("campaign", "session has done")
			v.runAddressConfiguration("delete")
			break
		case <-v.ctx.Done():
			ctxTmp, cancelTmp := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(v.conf.Interval*5))
			err := e.Resign(ctxTmp)
			if err != nil {
				log.Println("Resign leader error. ")
				cancelTmp()
				log.Fatal(err)
				return
			}
			err = s.Close()
			if err != nil {
				cancelTmp()
				log.Println("Session close error. ")
				log.Fatal(err)
				return
			}
			cancelTmp()
			return
		}
	}
}

func (v *VipKeeper) runAddressConfiguration(action string) bool {
	cmd := exec.Command("ip", "addr", action,
		v.getCIDR(),
		"dev", v.netIface.Name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			log.Printf("Got error %s", output)
			return false
		}
		log.Printf("Error running ip address %s %s on %s: %s",
			action, v.vip, v.netIface.Name, err)
		return false
	}
	return true
}

func (v *VipKeeper) getCIDR() string {
	return fmt.Sprintf("%s/%d", v.vip.String(), netmaskSize(v.vipMask))
}
