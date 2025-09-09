package random

import (
	"github.com/sony/sonyflake"
)

var flake = sonyflake.NewSonyflake(sonyflake.Settings{})

func Snoyflake() uint64 {
	id, _ := flake.NextID()
	return id
}
