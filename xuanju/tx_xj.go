package xuanju

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"os"
	"os/signal"
	"syscall"
)

/*
 * 发起竞选
 * 未当选leader前，会一直阻塞在Campaign调用
 * 当选leader后，等待SIGINT、SIGTERM或session过期而退出
 * https://github.com/etcd-io/etcd/blob/master/etcdctl/ctlv3/command/elect_command.go
 */

func campaign(c *clientv3.Client, election string, prop string) error {
	//NewSession函数中创建了一个lease，默认是60s TTL，并会调用KeepAlive，永久为这个lease自动续约（2/3生命周期的时候执行续约操作）
	s, err := concurrency.NewSession(c)
	if err != nil {
		return err
	}
	e := concurrency.NewElection(s, election)
	ctx, cancel := context.WithCancel(context.TODO())

	donec := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		cancel()
		close(donec)
	}()

	//竞选逻辑，将展开分析
	if err = e.Campaign(ctx, prop); err != nil {
		return err
	}

	// print key since elected
	resp, err := c.Get(ctx, e.Key())
	if err != nil {
		return err
	}
	display.Get(*resp)

	select {
	case <-donec:
	case <-s.Done():
		return errors.New("elect: session expired")
	}

	return e.Resign(context.TODO())
}

/*
 * 类似于zookeeper的临时有序节点，etcd的选举也是在相应的prefix path下面创建key，该key绑定了lease并根据lease id进行命名，
 * key创建后就有revision号，这样使得在prefix path下的key也都是按revision有序
 * https://github.com/etcd-io/etcd/blob/master/clientv3/concurrency/election.go
 */

func (e *Election) Campaign(ctx context.Context, val string) error {
	s := e.session
	client := e.session.Client()

	//真正创建的key名为：prefix + lease id
	k := fmt.Sprintf("%s%x", e.keyPrefix, s.Lease())
	//Txn：transaction，依靠Txn进行创建key的CAS操作，当key不存在时才会成功创建
	txn := client.Txn(ctx).If(v3.Compare(v3.CreateRevision(k), "=", 0))
	txn = txn.Then(v3.OpPut(k, val, v3.WithLease(s.Lease())))
	txn = txn.Else(v3.OpGet(k))
	resp, err := txn.Commit()
	if err != nil {
		return err
	}
	e.leaderKey, e.leaderRev, e.leaderSession = k, resp.Header.Revision, s
	//如果key已存在，则创建失败；
	//当key的value与当前value不等时，如果自己为leader，则不用重新执行选举直接设置value；
	//否则报错。
	if !resp.Succeeded {
		kv := resp.Responses[0].GetResponseRange().Kvs[0]
		e.leaderRev = kv.CreateRevision
		if string(kv.Value) != val {
			if err = e.Proclaim(ctx, val); err != nil {
				e.Resign(ctx)
				return err
			}
		}
	}

	//一直阻塞，直到确认自己的create revision为当前path中最小，从而确认自己当选为leader
	_, err = waitDeletes(ctx, client, e.keyPrefix, e.leaderRev-1)
	if err != nil {
		// clean up in case of context cancel
		select {
		case <-ctx.Done():
			e.Resign(client.Ctx())
		default:
			e.leaderSession = nil
		}
		return err
	}
	e.hdr = resp.Header

	return nil
}
