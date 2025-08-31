package order_contract

type CreateReq struct {
	UserId  int
	GoodsId int
}
type CreateRes struct {
	OrderId int
}
