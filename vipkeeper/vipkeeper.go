package vipkeeper

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"log"
	"net"
	"os"
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
		fmt.Println("elect: success")
		//TODO Bind VIP here
		select {
		case <-s.Done(): // 如果因为网络因素导致与etcd断开了keepAlive，这里break，重新创建session，重新选举
			log.Println("campaign", "session has done")
			//TODO Release VIP here
			break
		case <-v.ctx.Done():
			ctxTmp, _ := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(v.conf.Interval))
			e.Resign(ctxTmp)
			s.Close()
			return
		}
	}
}
