package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/urfave/cli"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

func doMain(opt *cli.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 接続先のURLをパース
	u, err := url.Parse(opt.String("url"))
	if err != nil {
		log.Fatal(err)
	}

	// ログインユーザーとパスワードをセット
	u.User = url.UserPassword(opt.String("user"), opt.String("password"))

	// ログイン
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatal(err)
	}

	// ViewManager作成
	m := view.NewManager(c.Client)

	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"ClusterComputeResource"}, true)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Destroy(ctx)

	var cls []mo.ClusterComputeResource
	if err = v.Retrieve(ctx, []string{"ClusterComputeResource"}, []string{"host", "name"}, &cls); err != nil {
		log.Fatal(err)
	}

	pc := property.DefaultCollector(c.Client)
	var hosts []mo.HostSystem
	for _, cl := range cls {
		if cl.Name == opt.String("cluster") {
			clRefArray := cl.Host
			err := pc.Retrieve(ctx, clRefArray, []string{"vm", "name"}, &hosts)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	var vms []mo.VirtualMachine
	for _, host := range hosts {
		vmRefArray := host.Vm
		err := pc.Retrieve(ctx, vmRefArray, []string{"config"}, &vms)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Memory Reservation and Limit.
	var totalReservation int64
	var totalLimit int64
	for _, vm := range vms {
		if totalReservation == 0 {
			totalReservation = *vm.Config.MemoryAllocation.Reservation
		} else {
			totalReservation = totalReservation + *vm.Config.MemoryAllocation.Reservation
		}

		if totalLimit == 0 && *vm.Config.MemoryAllocation.Limit >= 0 {
			totalLimit = *vm.Config.MemoryAllocation.Limit
		} else if *vm.Config.MemoryAllocation.Limit >= 0 {
			totalLimit = totalLimit + *vm.Config.MemoryAllocation.Limit
		}
	}
	fmt.Println(totalReservation)
	fmt.Println(totalLimit)
}

func main() {
	app := cli.NewApp()
	app.Name = "example"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url",
			Value: "https://127.0.0.1/sdk",
			Usage: "vCenter/ESXiのSDK URL",
		},
		cli.StringFlag{
			Name:  "user, u",
			Value: "administrator@vsphere.local",
			Usage: "ログインユーザー名",
		},
		cli.StringFlag{
			Name:  "password, p",
			Usage: "ログインパスワード",
		},
		cli.StringFlag{
			Name:  "cluster, c",
			Usage: "予約状況を確認するクラスタ名",
		},
	}
	app.Action = doMain
	app.Run(os.Args)
}
