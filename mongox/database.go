package mongox

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var db *mongo.Database

// InitDB 项目启动先初始化DB
func InitDB(uri string, dbname string) error {
	// 设置 MongoDB 连接选项
	clientOptions := options.Client().ApplyURI(uri).SetBSONOptions(&options.BSONOptions{
		UseLocalTimeZone: true,
		DefaultDocumentM: true,
	})

	// 连接到 MongoDB
	client, err := mongo.Connect(clientOptions)
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
