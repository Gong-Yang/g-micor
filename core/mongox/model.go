package mongox

import (
	"github.com/Gong-Yang/g-micor/core/util/random"
	"time"
)

type Base struct {
	Id         uint64    `bson:"_id,omitempty"`
	CreateTime time.Time `bson:"createTime"`
	UpdateTime time.Time `bson:"updateTime"`
}

func (b *Base) GetId() any {
	return b.Id
}

func (b *Base) Create() {
	b.Id = random.Snoyflake()
	b.CreateTime = time.Now()
	b.UpdateTime = time.Now()
}
