package mongox

import (
	"context"
	"github.com/Gong-Yang/g-micor/core/syncx"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
)

var collMap = syncx.NewResourceManager[mongo.Collection]()

type CollInterface interface {
	Create()
	Update()
}

type Coll[T CollInterface] struct {
	CollName string
	Indexes  []mongo.IndexModel
}

func (t *Coll[T]) FindOne(ctx context.Context, filter interface{},
	opts ...*options.FindOneOptions) (res *T, err error) {
	coll := getColl(ctx, t)
	err = coll.FindOne(ctx, filter, opts...).Decode(&res)
	return
}

func (t *Coll[T]) InsertOne(ctx context.Context, document interface{},
	opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	coll := getColl(ctx, t)
	if c, ok := document.(CollInterface); ok {
		c.Create()
	}
	return coll.InsertOne(ctx, document, opts...)
}

func (t *Coll[T]) GetColl(ctx context.Context) *mongo.Collection {
	return getColl(ctx, t)
}

// 获取集合
func getColl[T CollInterface](ctx context.Context, coll *Coll[T]) *mongo.Collection {
	resource, _ := collMap.GetResource(coll.CollName, func() (*mongo.Collection, error) {
		// 创建集合
		collection := db.Collection(coll.CollName)
		if len(coll.Indexes) == 0 {
			return collection, nil
		}
		// 创建索引
		_, err := collection.Indexes().CreateMany(ctx, coll.Indexes)
		if err != nil {
			slog.ErrorContext(ctx, "init coll index error", "collection", coll.CollName, "error", err)
		} else {
			slog.InfoContext(ctx, "init coll index", "collection", coll.CollName)
		}
		return collection, nil
	})
	return resource
}
