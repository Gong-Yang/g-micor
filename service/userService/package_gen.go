package userService

type Service struct {
}

func (s Service) Register(req *user.RegisterReq, res *user.RegisterRes) (err error) {
	res_, err := Register(req)
	if err != nil {
		return
	}
	*res = *res_
	return
}
