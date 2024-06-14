- 第一天：**服务端与消息编码**，使用encoder和decoder来进行编解码，首先用发送option字段通知服务端，要使用哪种编解码器。然后发送Header-body格式消息，header中会有访问的服务和seq序列号，body里有访问的参数。
> 用一个map，key是编码器的类型（例如：Json、Gob、Yaml），value是New函数，根据传入的编码器类型不同，所用的构造函数也不同（例如：NewJson、NewGob、NewYaml）
- 第二天：**高性能客户端**，使用call来进行封装，表示一次rpc调用会话，包含了访问的服务方法、seq、参数、返回值。客户端首先建立TCP连接，发送option，根据返回的套接字conn构建rpc client，调用client call方法将访问的服务方法、参数封装起来，编码后发送给服务端，服务端在接收到后解码，然后构造响应头和包体，包体由调用的方法决定，然后编码发回客户端。客户端收到后解码，放入reply中。
- 第三天：**服务注册**，一个服务器上会有很多的服务，很多方法，因此需要事先注册。服务是从结构体映射过来的，方法就是结构体的方法，在这里通过反射，可以获得结构体的value和type，通过type可以获取到结构体拥有的方法，包括方法的数量和入参回参。 
```go
var s Student
typ := reflect.Typeof(s)
NumMethod := typ.NumMethod()
method := typ.Method(1)
```
> 方法属于结构体的类型，且方法名要大写，才可以被发现。
 ```go
func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}
```
在服务器端通过map保存服务注册的结果，用sync.Map去存。

- 第四天：**超时处理**，超时就是通过监听channel来完成的，用一个goroutine去完成业务，如果业务完成，向ch中传入信号。主线程在等待，利用select监听time.after和ch，如果从time.after里读取到，说明超时了，如果从ch读到说明没超时。
> 这里借助time.after，会返回一个channel，如果超过了给定时间，会从这个channel里面返回信号。(定时器)
```go
ch := make(chan clientResult)
go func() {
	client, err := newClient(conn, opt)
	ch <- clientResult{
		client: client,
		err:    err,
	}
}()
if opt.ConnectTimeout == 0 {
    result := <-ch
    return result.client, result.err
}
select {
case <-time.After(opt.ConnectTimeout):
	return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
case result := <-ch:
	return result.client, result.err
}
```
`opt.ConnectTimeout == 0 `说明可以无限等待
- 第五天：**支持HTTP协议**，Web 开发中，我们经常使用 HTTP 协议中的 GET、POST 等方式发送请求，但 RPC 的消息格式与标准的 HTTP 协议并不兼容，在这种情况下，就需要一个协议的转换过程。HTTP 协议的 CONNECT 方法恰好提供了这个能力，CONNECT 一般用于代理服务。
也就是说，首先用CONNECT方法建立连接，连接建立后用hijack把HTTP连接劫持下来，变成TCP连接，在这个TCP连接上进行RPC消息的传输。客户端和服务端都要实现HTTP的处理，客户端发送CONNECT，服务端接收到CONNECT后，劫持下来，然后进行RPC。服务端就是要实现http.handler接口，也就是实现serverHTTP方法。
> hijack将HTTP连接变成TCP连接，在这个连接上传递消息就不用遵循HTTP格式了。

- 第六天：**负载均衡**。常用的负载均衡算法有：
> **随机选择策略**：从服务列表中随机选择一个。

> **轮询算法(Round Robin)**：依次调度不同的服务器，每次调度执行 i = (i + 1) mode n。
 
> **加权轮询(Weight Round Robin)**：在轮询算法的基础上，为每个服务实例设置一个权重，高性能的机器赋予更高的权重，也可以根据服务实例的当前的负载情况做动态的调整，例如考虑最近5分钟部署服务器的 CPU、内存消耗情况。

> **哈希/一致性哈希策略**：依据请求的某些特征，计算一个 hash 值，根据 hash 值将请求发送到对应的机器。一致性 hash 还可以解决服务实例动态添加情况下，调度抖动的问题。一致性哈希的一个典型应用场景是分布式缓存服务。【GeeCache】

在这里实现了随机和轮询方法：
```go
switch mode {
case RandomSelect:
    return m.servers[m.r.Intn(n)], nil
case RoundRobinSelect:
    index := m.index % n
    m.index = index + 1
    return m.servers[index], nil
}
```

第七天：服务发现与注册中心，在第六天我们是通过硬编码服务器地址告诉客户端访问RPC服务时可以去这几个服务器，但是我们不应该这样。为此，我们实现一个注册中心的结构，服务端会定时向注册中心上报自己的地址，客户端会定期从注册中心拿现存的服务器地址，然后选择一个发送RPC请求。注册中心就是一个HTTP服务端和HTTP客户端，接收HTTP消息，在HTTP头部中放入server的地址。
服务端需要定期向注册中心发送心跳heartbeat：
```go
t := time.NewTicker(duration)
for err == nil {
	<-t.C
	err = sendHeartbeat(registry, addr)
}
```
> time.NewTicker提供可以一直使用的定时器，负责定期任务。time.after提供一次性定时器，负责一次性任务。