# **Golang Context 包详解与使用**
## **1 使用背景**
## **2 context 的简单demo**
## **3 context 的数据结构**
## **4 context 的实际使用**
## **5 context 的内部实现**
## **6 参考**

## **1 使用背景**
Golang 的并发调度模型(M:N)是十分高效的，但这也随之而来带入了一个问题，如何控制协程之间的并发呢？<br>
一共存在三种方式：
>1) sync 包的 waitgroup
>2) channel 控制
>3) context 控制

1) 首先对于 sync.waitgroup，它只能在量上建立控制关系。在一个 goroutine 下，Add 了多少协程，协程结束后又 Done 了多少，最后 wait 等待； <br>
2) 其次对于 channel 控制，它可以针对特定的 goroutine 进行控制， 这只需要我们使用指定的 channel 变量即可。但是随着 goroutine 越多，尤其是 goroutine 所产生的新的 goroutine ，新的 goroutine 又产生另外的新的 goroutine(这样的调用链就类似于树状的结构了)，这种情况需要怎么管理呢？
这样用 channel 管理起来就略显不足了。要是有一种能随着 goroutine 的调用链的扩展而扩展的监控就好了。<br>
>ps: goroutine中并无父子关系，也就没有所谓子进程退出后的通知机制，goroutine都是被平行调度的(调度的原则后面的一篇文章会分享)，上述说的“类似树状结构的调用链”比较迎合 context 功能描述，毕竟在 context 的结构中，确实是有父子关系的。
3) 恰好，context 就提供了这样一种机制，它是一种跟踪 goroutine 调用链的方案。<br>
3.1) context 库的设计目的就是跟踪 goroutine 调用链，并在这些 gouroutine 调用链中传递通知和元数据。两个目的：<br>
(1) 退出通知机制：通知可以传递给整个 goroutine 调用链上的每一个 goroutine。<br>
(2) 传递数据：数据可以传递给整个 goroutine 调用链上的每一个 goroutine。<br>

## **2 context 的简单demo**
### **2.1 控制一个 goroutine**
程序如下：
```
package main

import (
	"fmt"
	"context"
	"time"
)

// 传入 ctx， 当没有结束信号时，monitor1 一直执行，否则退出
func monitor1(ctx context.Context)  {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("goroutine exit...")
			return
		default:
			fmt.Println("goroutine running...")
			time.Sleep(1 * time.Second)
		}
	}
}

func main()  {
	ctx, cancel := context.WithCancel(context.Background())
	go monitor1(ctx)
	time.Sleep(5 * time.Second)
	fmt.Println("Ready to exit")
	cancel()
	// 3s 内没有输出 "goroutine running..." 则说明 monitor1 已经停止
	time.Sleep(3 * time.Second)
    fmt.Println("main stop")
}
```
输出如下：
```
nikkoyang@nikkoyang-LB0:~/GoProj/src/GolangMarkdown/Context$ go run main.go 
goroutine running...
goroutine running...
goroutine running...
goroutine running...
goroutine running...
Ready to exit
goroutine exit...
main stop
```

### **2.2 并行控制多个 goroutine**
```
package main

import (
	"fmt"
	"context"
	"time"
)

// 传入 ctx， 当没有结束信号时，monitor1 一直执行，否则退出
func monitor1(ctx context.Context, id int)  {
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
	go monitor1(ctx, 2)
	go monitor1(ctx, 3)

	time.Sleep(3 * time.Second)
	fmt.Println("Ready to exit")
	cancel()
	// 3s 内没有输出 "goroutine running..." 则说明 monitor1 已经停止
	time.Sleep(3 * time.Second)
    fmt.Println("main stop")
}
```
输出代码基本同上：
```
nikkoyang@nikkoyang-LB0:~/GoProj/src/GolangMarkdown/Context$ go run main.go 
goroutine [1] running...
goroutine [2] running...
goroutine [3] running...
goroutine [1] running...
goroutine [3] running...
goroutine [2] running...
goroutine [3] running...
goroutine [1] running...
goroutine [2] running...
Ready to exit
goroutine [2] exit...
goroutine [1] exit...
goroutine [3] exit...
main stop
```

## **3 context 的数据结构**
### **3.1 context 的工作机制**
context 可以比 sync 和 chaneel 适用于复杂的并发模型是因为它能够自行构建出树状模型来监控 goroutine 的调用链。<br>
第一个创建 Context 的 goroutine 被称为 root 节点。root 节点负责创建一个实现 Context 接口的具体对象，并将该对象作为参数
传递到其新拉起的 goroutine，下游的 goroutine 可以继续封装该对象(形成调用链)，再传递到更下游的 goroutine。Context 对象在传递的过程中最终形成一个树状的数据结构(整个调用链)，这样通过位于 root 节点(树的根节
点)的 Context 对象就能遍历整个 Context 对象树，通知和消息就可以通过 root 节点传递至下游，实现了上游 goroutine 对下游 goroutine 的消息传递。
>调用链必须保持，否则无法对中断处以下的 goroutine 进行通知。

### **3.2 context 函数接口**
context 涉及的函数接口和类型定义比较少，一共就这么几个，其中WithXXX函数是使用 context 包时，需要直接关注的：
```
type CancelFunc
type Context
type cancelCtx struct
type timerCtx struct
type valueCtx struct
func Background() Context
func TODO() Context
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc)
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
func WithValue(parent Context, key, val interface{}) Context
```
下面逐一分析：
1. Context接口
```
type Context interface {
    // 如果 Context 实现了超时控制，则该方法返回 ok true, deadline 为超时时间，
    // 否则 ok 为 false
    Deadline() (deadline t 工me . Time , ok bool)

    // Done() 是一个只读的方法，接收该 Context 对象的 goroutine 应该监听该方法返回的 chan ，以便及时释放资源
    Done() <-chan struct{}

    // Done 返回的 chan 收到通知的时候，才可以访问 Err()获知因为什么原因被取消
    Err() error

    // 可以访问上游 goroutine 传递给下游 goroutine 的位
    Value(key interface{}) interface{}
}
```

1.1) 查看源码 context.go 文件的 line 200 左右，可以看到系统已经定义了两个全局变量也就是全局的 parent context，这两个差不过，一般都用第一个：
```
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)
```
分别通过以下函数来调用：
```
func Background() Context {
	return background
}

func TODO() Context {
	return todo
}
```
可见，backgroup 和 todo 都是一个 emptyCtx 指针对象，这个 emptyCtx 指针也就实现了 Context 的接口，只是里面什么都没有做。等用户去调用 background 再去自行实现：
```
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}
```

2. 源码中 Context 接口的实现： <br>
源码中一共有三个结构体类型实现了这个 Context 接口： cancelCtx、timerCtx、valueCtx。后续的功能代码也都是围绕这三个结构体进行拓展<br>

2.1) cancelCtx 主要实现了 Context 和 canceler 接口。作用是删除自身的同时，通知订阅了自己的子节点所在的 goroutine，整个 cancelCtx 相关的方法如下：
```
// A cancelCtx can be canceled. When canceled, it also cancels any children
// that implement canceler.
type cancelCtx struct {
	Context

	mu       sync.Mutex            // 访问的互斥锁
	done     chan struct{}         // 往 done 中填充数据，以让后续的 context 接收，由第一个 cancel 发起者调用
	children map[canceler]struct{} // 保存子 context
	err      error                 // 第一个调用 cancel 的 context 的 err 非空。
}

// Done() 方法实现了往 done 中填充数据
func (c *cancelCtx) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *cancelCtx) String() string {
	return fmt.Sprintf("%v.WithCancel", c.Context)
}


// 关闭 c.done，若 removeFromParent 为 true，则把 c 从其父 context 的调用链中删除
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
    // err 非空，即 panic
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	if c.done == nil {
		c.done = closedchan
	} else {
		close(c.done)
	}

    // 循环调用子 context 的 cancel, 与由于它们的父节点(c)在后面的代码中已经被从调用链中删除，
    // 所以子 context 只需要通知自身和自身以下的子节点即可。
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

    // removeFromParent 为 true，则把自身从调用链中删除。
	if removeFromParent {
		removeChild(c.Context, c)
	}
}
```
以上便是 cancelCtx 的主要实现。这里有两点注意的(第4小节介绍)：
1) err 为空即 panic ，如何非空？
2) removeFromParent 部分是怎么把自己从调用链删除的？

2.2) timerCtx 主要是实现了 Context 接口，并在内部封装了 cancelCtx 类型实例 ，同时有一个 deadline，用来实现定时退出通知：

```
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) String() string {
	return fmt.Sprintf("%v.WithDeadline(%s [%s])", c.cancelCtx.Context, c.deadline, time.Until(c.deadline))
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {

    // 对应上面的 (c *cancelCtx) cancel() 方法，删除自身以下的子节点
	c.cancelCtx.cancel(false, err)

    // 是否把自身移出调用链
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
```

2.3) valueCtx 在内部封装了 Context 接口类型，同时集成了一个 k/v 的存储变量，可以存储任意类型。 valueCtx 可用来传递通知信息。可见在 valueCtx 中并没有集成 cancel() 方法：
```
type valueCtx struct {
	Context
	key, val interface{}
}

func (c *valueCtx) String() string {
	return fmt.Sprintf("%v.WithValue(%#v, %#v)", c.Context, c.key, c.val)
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
```

2.4) 几个直接使用的函数：
// 创建一个带有退出通知的 Context，内部有 cancelCtx 实例，实现了 cancel 方法 <br>
func WithCancel (parent Context) (Context, cancel CancelFunc)

// 创建一个带有超时通知的 Context，内部有 timerCtx 实例，实现了 cancel 方法 <br>
func WithDeadline (parent Context , deadline t ime . Time ) (Context, Cancel Func)

// 创建一个带有超时通知的 Context，内部有 timerCtx 实例，实现了 cancel 方法 <br>
// WithTimeOut 其实就是 WithDeadline 返回的。
func WithTimeout (parent Context , timeout time . Duration) (Context, Cancel Func )

// 创建一个带有传递数据的 Context，内部有 valueCtx 实例，无 cancel 方法 <br>
func WithValue(parent Context , key , val interface{}) Context

## **4 context 的实际使用**
下面来看一下 context 控制一条goroutine 调用链的实际使用，再深究一下内部的实现逻辑：

```
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
	go monitorCancel(ctx1, "ctx1")

	// 设置 ctx2 超时时间为 2 秒，2 秒后会触发退出通知
	tm := 2 * time.Second
	ctx2, _ := context.WithTimeout(ctx1, tm)
	go monitorTimeout(ctx2, "ctx2")

	ctx3 := context.WithValue(ctx2, "k3", "monitorValue")
	go monitorValue(ctx3, "ctx3")

	// 等待 4 秒，查看 ctx2 和 ctx3 退出，以及 monitorCancel 的继续运行
	time.Sleep(4 * time.Second)

	// ctx1 退出
	cancel()

	time.Sleep(3 * time.Second)
	fmt.Println("main stop")
}
```
查看下面的输出，可以发现， ctx2 触发超时后，通知了 ctx3 的同时，把自己也从 ctx1 发起的调用链删除了：
```
ctx3 is running, and value is monitorValue
ctx1 is running
ctx2 is running
ctx3 is running, and value is monitorValue
ctx2 is running
ctx1 is running
ctx3 receive exit signal                 (ctx3 退出)
ctx2 receive exit signal                 (ctx2 退出)
ctx1 is running                         
ctx1 is running                          (monitorCancel 继续运行 2 秒)
ctx1 receive exit signal
main stop
```
若把超时时间 tm(5秒) 设置的比 ctx1 cancel() 前的延时长(2秒)，则 ctx1 退出后，会触发 ctx2，ctx3一同退出：
```
ctx3 is running, and value is monitorValue
ctx1 is running
ctx2 is running
ctx3 is running, and value is monitorValue
ctx2 is running
ctx1 is running
ctx2 receive exit signal
ctx1 receive exit signal
ctx3 receive exit signal
main stop
```

## **5 context 的内部实现**
回到之前的问题，再根据第4小节的用例，总结以下几点：
1) 在 cancel 的时候， err 非空？
2) 是通过怎么样的方式找到 子 context 的？
3) 是怎么把自己从调用链中删除的？
<br><br>

以 WithCancel 为例来讨论：
当创建一个 ctx1 时，若创建成功，则会返回一个 Context 类型的 c, 和 CancelFunc 类型的函数：
```
ctx1, cancel := context.WithCancel(context.Background())

func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
    // 首先根据 context.Bacground() 创建一个 ctx1 的根 context
	c := newCancelCtx(parent)

    // 
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
```

此时作为返回的节点 ctx1 的 cancel 就是一个带有 (true, Canceled) 参数的方法。再回看前面的 cancelCtx 和 timerCtx 类型的 cancel 方法，
可以发现，分别对应 RemoveFromParent(true)， err (Canceled)。而 Canceled 就是已经定义好的全局 error 变量：
```
var Canceled = errors.New("context canceled")
```
所以 ctx1 创建成功，就会成为 root 节点，并当 cancel 被调用时候，就会通知子 context 并把自身删除掉 <br>

这里需要用到下面几个函数：
```
func propagateCancel(parent Context, child canceler) {
	if parent.Done() == nil {
		return // parent is never canceled
	}
	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
	
		}()
	}
}
```
propagateCancel 主要做了以下事情：
1) 判断 parent 的 Done() 是否为 nil。如果是，则说明 parent(context.Background()) 不是一个可取消的 Context 对象，也就无所谓取消构造树，说明 child (ctx1)就是取消构造树的根；
2) 如果 parent 的方法 Done() 返回值不是 nil，则向上回溯自己的祖先，找到第一个是 cancelCtx 类型实例的 context 节点，并将 child 的子节点注册维护到那棵关系树里面；
3) 如果向上回溯自己的祖先都不是 cancelCtx 类型实例，则说明整个 context 树条的取消树是不连续的。此时只需监听 parent 和自己的取消信号即可。

再来看 parentCancelCtx 是如何找到第一个有 cancelCtx 实例的 Context 节点的：
```
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	for {
		switch c := parent.(type) {
		case *cancelCtx:
			return c, true
		case *timerCtx:
			return &c.cancelCtx, true
		case *valueCtx:
			parent = c.Context
		default:
			return nil, false
		}
	}
}

func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}

func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	…………
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
    …………
}

func WithValue(parent Context, key, val interface{}) Context {
    ……
	return &valueCtx{parent, key, val}
}

==============================================================
    ctx1 = &cancelCtx{
        Context: context.Background()
    }
    ctx2 = &timerCtx{
        cancelCtx: ctx1,
        deadline:  d,
    }
    ctx3 =  valueCtx {
        Context: ctx2
        key, val interface{}
   }
```

这里可以看出来，parentCancelCtx 对 parenet 一直往上回调，若是 *cancelCtx 或 *timerCtx 即找到了最近的节点，若是 valueCtx(前面提到了，它没有取消能力) 则一直再往上查找。 <br>
这里也可以看出来，即使是由 ctx3 再派生出 ctx4，其实 ctx3 和 ctx4 的取消关系是平级的，都来源于 ctx2。 <br>
现在可见的是，创建 context 节点的时候，维护出来的 context 树的关系的双向的，双向的，双向的！！！：
1) ctx1 取消通知时，可以沿着 context 树一直找到 ctx2, ctx2 再找到 ctx3，这是通过 propagateCancel 把各个 context 子节点注册到父节点的 map(children) 中； 
2) 由于 cancelCtx, timerCtx, valueCtx 结构关系，它们的 Context 保存的就是父节点，所以可以根据 parentCancelCtx 找到所属的父节点，当然后续也就可以调用 removeChild(c.Context, c) 把从调用链删除 delete(p.children, child)。
```
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}
```
所以，从第4小节的用例来看:<br>
在可取消的调用链上： <br>
　　ctx1.Children ---> ctx2.Children ---> ctx3
在数据结构的引用链上：<br>
　　ctx3.Context ---> ctx2    <br>
　　ctx2.Context ---> ctx1    <br>
　　ctx1.Context ---> context.Background()    <br>


## **6 参考**
其实自己一开始也知道用一点 context 包，但是实际看源码的时候，如果没有资料的帮助发现还是很费劲的，尤其是自己都看不太清楚还要来码字的情况下，具体的借鉴参考如下：
1) 《Go语言核心编程》
2) [Go语言实战笔记（二十）| Go Context](https://www.flysnow.org/2017/05/12/go-in-action-go-context.html)
3) [官方源码](https://golang.org/pkg/context/)
