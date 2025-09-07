package store

import (
	"github.com/Gong-Yang/g-micor/core/mongox"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var UserStore = &userStore{CollName: "user"}

type User struct {
	Id primitive.ObjectID
}

type userStore mongox.Coll[User]
