package ginx

// 响应码对象
type Response struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}
