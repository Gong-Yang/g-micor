package user_contract

type RegisterReq struct {
	Name string
}

type RegisterRes struct {
	Id   int
	Name string
}
