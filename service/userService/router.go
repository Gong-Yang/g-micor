package userService

import (
	"github.com/Gong-Yang/g-micor/contract/user"
	"github.com/Gong-Yang/g-micor/core/ginx"
	"github.com/gin-gonic/gin"
)

func (s Service) Router(router gin.IRouter) {
	group := router.Group("/user")
	ginx.POST(group, "/register", s.Register, ginx.Body[user.RegisterReq]())
	return
}
