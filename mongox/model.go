package mongox

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ContextTenantId = "tenantId"

type Base struct {
	Id         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TenantId   string             `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	UpdateTime time.Time          `bson:"updateTime" json:"updateTime"`
}

func (b *Base) GetId() any {
	return b.Id
}

func (b *Base) Write(ctx context.Context) {
	if b.Id.IsZero() {
		b.Id = primitive.NewObjectID()
	}
	b.UpdateTime = time.Now()
	if b.TenantId == "" {
		b.setTenantId(ctx)
	}
}

func (b *Base) setTenantId(ctx context.Context) {
	// 从上下文中获取， 如果有则赋值
	if ctx == nil {
		return
	}
	if tenantId, ok := ctx.Value(ContextTenantId).(string); ok {
		b.TenantId = tenantId
	}
}

func NewBase(ctx context.Context) *Base {
	res := &Base{}
	res.Write(ctx)
	return res
}
