package ginx

// OpenRouter 默认公开接口
func OpenRouter() *RouterConf {
	return &RouterConf{
		timeOut:   DefaultTimeOut,
		sinWay:    SignWayNormal,
		needLogin: false,
	}
}

func LoginRouter(appid string, author Author) *RouterConf {
	return &RouterConf{
		timeOut:   DefaultTimeOut,
		sinWay:    SignWayNormal,
		needLogin: true,
		appId:     appid,
		author:    author,
	}
}
