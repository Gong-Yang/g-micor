package mongox

import (
	"encoding/binary"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var Convert = convert{}

type convert struct {
}

func (c convert) TimeToObjectId(date time.Time) bson.ObjectID {
	var b [12]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(date.Unix()))
	return b
}
