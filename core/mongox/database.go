package mongox

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"time"
)

var db *mongo.Database

// InitDB 项目启动先初始化DB
func InitDB(uri string, dbname string) error {
	// 设置 MongoDB 连接选项
	clientOptions := options.Client().ApplyURI(uri).SetRegistry(newRegistry())

	// 连接到 MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		fmt.Println("error connecting to mongodb,err:", err)
		return err
	}

	// 检查连接
	err = client.Ping(context.Background(), nil)
	if err != nil {
		fmt.Println("error pinging mongodb,err:", err)
		return err
	}

	fmt.Println("Connected to MongoDB successfully!")
	db = client.Database(dbname)

	return nil
}

// 新版注册函数 TODO 质疑
func newRegistry() *bsoncodec.Registry {
	registry := bson.NewRegistry()
	registry.RegisterTypeDecoder(reflect.TypeOf(time.Time{}), bsoncodec.ValueDecoderFunc(decodeTime))
	return registry
}

// 时间解码函数
func decodeTime(dCtx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	// 这里你可以自定义解码逻辑
	// 读取 BSON 中的 DateTime 并转换为 time.Time
	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}
	// 设置时间为当前服务器时区
	val.Set(reflect.ValueOf(time.UnixMilli(dt).In(time.Local)))
	return nil
}
