package ginx

// OpenRouter 默认公开接口
func OpenRouter() *RouterConf {
	return &RouterConf{
		timeOut:   DefaultTimeOut,
		sinWay:    "",
		needLogin: false,
	}
}

func LoginRouter(appid string, author Author) *RouterConf {
	return &RouterConf{
		timeOut:   DefaultTimeOut,
		sinWay:    "",
		needLogin: true,
		appId:     appid,
		author:    author,
	}
}
