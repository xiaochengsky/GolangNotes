package main

import (
	"context"
	"fmt"
	"time"
)

type C interface {
	get() (int)
}

type A struct {
	C
	num int
}

/*
// 传入 ctx， 当没有结束信号时，monitor1 一直执行，否则退出
func monitor1(ctx context.Context, id int)  {
	ctx1, _ := context.WithCancel(ctx)
	go monitor2(ctx1, id+1)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("goroutine [%d] exit...\n", id)
			return
		default:
			fmt.Printf("goroutine [%d] running...\n", id)
			time.Sleep(1 * time.Second)
		}
	}
}

func monitor2(ctx context.Context, id int)  {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("goroutine [%d] exit...\n", id)
			return
		default:
			fmt.Printf("goroutine [%d] running...\n", id)
			time.Sleep(1 * time.Second)
		}
	}
}

func main()  {
	ctx, cancel := context.WithCancel(context.Background())
	go monitor1(ctx, 1)
	time.Sleep(3 * time.Second)
	fmt.Println("Ready to exit")
	cancel()
	// 3s 内没有输出 "goroutine running..." 则说明 monitor1 已经停止
	time.Sleep(3 * time.Second)

}
 */

func monitorCancel(c context.Context, name string)  {
	for {
		select {
		case <- c.Done():
			fmt.Printf("%s receive exit signal\n", name)
			return
		default:
			fmt.Printf("%s is running\n", name)
			time.Sleep(1 * time.Second)
		}
	}
}

func monitorTimeout(c context.Context, name string)  {
	for {
		select {
		case <- c.Done():
			fmt.Printf("%s receive exit signal\n", name)
			return
		default:
			fmt.Printf("%s is running\n", name)
			time.Sleep(1 * time.Second)
		}
	}
}

func monitorValue(c context.Context, name string)  {
	for {
		select {
		case <- c.Done():
			fmt.Printf("%s receive exit signal\n", name)
			return
		default:
			v := c.Value("k3").(string)
			fmt.Printf("%s is running, and value is %s\n", name, v)
			time.Sleep(1 * time.Second)
		}
	}
}

func main()  {

	ctx1, cancel := context.WithCancel(context.Background())
	fmt.Printf("ctx.Done = %v\n", context.Background().Done())
	fmt.Printf("ctx.Done = %v\n", ctx1.Done())
	go monitorCancel(ctx1, "ctx1")

	// 设置 ctx2 超时时间为 2 秒，2 秒后会触发退出通知
	tm := 2 * time.Second
	ctx2, _ := context.WithTimeout(ctx1, tm)
	go monitorTimeout(ctx2, "ctx2")

	ctx3 := context.WithValue(ctx2, "k3", "monitorValue")
	ctx4 := context.WithValue(ctx3, "k4", "monitorValue-k4")
	go monitorValue(ctx3, "ctx3")
	go monitorValue(ctx4, "ctx4")
	// 等待 3 秒，查看 ctx2 和 ctx3 退出
	time.Sleep(4 * time.Second)

	// ctx1 退出
	cancel()

	time.Sleep(3 * time.Second)
	fmt.Println("main stop")
}