package random

import (
	"fmt"

	"github.com/sony/sonyflake"
)

var flake = sonyflake.NewSonyflake(sonyflake.Settings{})

func Snoyflake() uint64 {
	id, _ := flake.NextID()
	return id
}
func SnoyflakeString() string {
	id, _ := flake.NextID()
	return fmt.Sprintf("%d", id)
}
