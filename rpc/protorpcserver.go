package main


import (
	"errors"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

type OrderService struct {

}

func (os *OrderService) GetOrderInfo(req *OrderRequest, resp *OrderInfo) error {

	// 已经存储的订单信息
	orderMap := map[string]OrderInfo{
		"000001": {OrderId: "orderId-01", OrderName: "衣服", OrderStatus: "已付款"},
		"000002": {OrderId: "orderId-02", OrderName: "零食", OrderStatus: "已付款"},
		"000003": {OrderId: "orderId-03", OrderName: "食品", OrderStatus: "未付款"},
	}

	orderId := req.OrderId

	current := time.Now().Unix()
	if req.TimeStamp > current {
		resp = &OrderInfo{OrderId: "0", OrderName: "", OrderStatus: "异常"}
	} else {
		if orderMap[orderId].OrderId != "" {
			*resp = orderMap[orderId]
		} else {
			return errors.New("server error")
		}
	}
	return nil
}

func RpcServer() {
	service := new(OrderService)

	err := rpc.RegisterName("S", service)
	if err != nil {
		panic(err)
	}

	rpc.HandleHTTP()

	listen, err := net.Listen("tcp", ":8080")
	err = http.Serve(listen, nil)
	if err != nil {
		panic(err)
	}
}
