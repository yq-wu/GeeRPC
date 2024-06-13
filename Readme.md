- 第一天：服务端与消息编码，使用encoder和decoder来进行编解码，首先用发送option字段通知服务端，要使用哪种编解码器。然后发送Header-body格式消息，header中会有访问的服务和seq序列号，body里有访问的参数。
> 用一个map，key是编码器的类型（例如：Json、Gob、Yaml），value是New函数，根据传入的编码器类型不同，所用的构造函数也不同（例如：NewJson、NewGob、NewYaml）
- 第二天：高性能客户端，使用call来进行封装，表示一次rpc调用会话，包含了访问的服务方法、seq、参数、返回值。客户端首先建立TCP连接，发送option，根据返回的套接字conn构建rpc client，调用client call方法将访问的服务方法、参数封装起来，编码后发送给服务端，服务端在接收到后解码，然后构造响应头和包体，包体由调用的方法决定，然后编码发回客户端。客户端收到后解码，放入reply中。
- 第三天：服务注册，一个服务器上会有很多的服务，很多方法，因此需要事先注册。服务是从结构体映射过来的，方法就是结构体的方法，在这里通过反射，可以获得结构体的value和type，通过type可以获取到结构体拥有的方法，包括方法的数量和入参回参。 
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