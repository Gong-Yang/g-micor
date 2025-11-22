package mongox

import (
	"context"
	"log/slog"

	"github.com/Gong-Yang/g-micor/syncx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collMap = syncx.NewResourceManager[mongo.Collection]()

type CollInterface interface {
	Write(ctx context.Context)
	GetId() any
	//SetTenantId(ctx context.Context)
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

func (t *Coll[T]) FindById(ctx context.Context, id interface{},
	opts ...*options.FindOneOptions) (res *T, err error) {
	coll := getColl(ctx, t)
	err = coll.FindOne(ctx, bson.M{"_id": id}, opts...).Decode(&res)
	return
}
func (t *Coll[T]) Find(ctx context.Context, filter interface{},
	opts ...*options.FindOptions) (res []*T, err error) {
	coll := getColl(ctx, t)
	cur, err := coll.Find(ctx, filter, opts...)
	if err != nil {
		return
	}
	err = cur.All(ctx, &res)
	return
}

func (t *Coll[T]) InsertOne(ctx context.Context, document interface{},
	opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	coll := getColl(ctx, t)
	if c, ok := document.(CollInterface); ok {
		c.Write(ctx)
	}
	return coll.InsertOne(ctx, document, opts...)
}
func (t *Coll[T]) InsertMany(ctx context.Context, documents []CollInterface,
	opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	coll := getColl(ctx, t)
	res := make([]interface{}, len(documents))
	for i, document := range documents {
		document.Write(ctx)
		res[i] = document
	}
	return coll.InsertMany(ctx, res, opts...)
}
func (t *Coll[T]) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	coll := getColl(ctx, t)
	return coll.UpdateOne(ctx, filter, update)
}
func (t *Coll[T]) BulkWrite(ctx context.Context, models []mongo.WriteModel,
	opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	coll := getColl(ctx, t)
	return coll.BulkWrite(ctx, models, opts...)
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

func (t *Coll[T]) SetById(ctx context.Context, obj CollInterface) (*mongo.UpdateResult, error) {
	obj.Write(ctx)
	return t.UpdateOne(ctx, bson.M{"_id": obj.GetId()}, bson.M{"$set": obj})
}
func (t *Coll[T]) UpsertManyById(ctx context.Context, documents []CollInterface) (*mongo.BulkWriteResult, error) {
	// 初始化
	var writers []mongo.WriteModel
	for _, document := range documents {
		document.Write(ctx)

		model := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": document.GetId()}).
			SetUpdate(bson.M{"$set": document}).
			SetUpsert(true)
		writers = append(writers, model)
	}
	return t.BulkWrite(ctx, writers)
}
