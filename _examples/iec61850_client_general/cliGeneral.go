package main

import (
	"fmt"
	"github.com/themeyic/go-iec61850/iec61850"
)

type myClient struct{}

func main() {
	var err error

	option := iec61850.NewOption()
	if err = option.AddRemoteServer("10.211.55.4:102"); err != nil {
		panic(err)
	}


	client := iec61850.NewClient( option)

	client.LogMode(true)

	client.SetOnConnectHandler(func(c *iec61850.Client) {
		c.SendStartDt() // 发送startDt激活指令
	})
	err = client.Start()

	for{
		select {
		case getValue := <-iec61850.Transfer :
			test := fmt.Sprintf("%v",getValue)
			fmt.Println("%x",test)
			return
		}
	}

	//if err != nil {
	//	panic(fmt.Errorf("Failed to connect. error:%v\n", err))
	//}
	//
	//for {
	//	time.Sleep(time.Second * 100)
	//}

}
