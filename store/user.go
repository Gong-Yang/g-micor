package store

import (
	"github.com/Gong-Yang/g-micor/core/mongox"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	*mongox.Base `bson:",inline"`
	UserName     string `bson:"userName,omitempty"`
}

var UserStore = &userStore{
	Coll: &mongox.Coll[User]{
		CollName: "user",
		Indexes: []mongo.IndexModel{
			{
				Keys:    bson.D{{"userName", 1}},
				Options: options.Index().SetUnique(true),
			},
		},
	},
}

type userStore struct {
	*mongox.Coll[User]
}
