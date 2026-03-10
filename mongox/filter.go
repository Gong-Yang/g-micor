package mongox

import "go.mongodb.org/mongo-driver/v2/bson"

type Filter bson.M

const (
	eq = "$eq"
	lt = "$lt"
	in = "$in"
)

func Eq(key string, value any) Filter {
	return Filter{key: bson.M{eq: value}}
}
func In(key string, value any) Filter {
	return Filter{key: bson.M{in: value}}
}
func Lt(key string, value any) Filter {
	return Filter{key: bson.M{lt: value}}
}

func (f Filter) set(opt, key string, value any) Filter {
	vmap, ok := f[key]
	if !ok {
		vmap = bson.M{}
		f[key] = vmap
	}
	vmap.(bson.M)[opt] = value
	return f
}

func (f Filter) Eq(key string, value any) Filter {
	return f.set(eq, key, value)
}
func (f Filter) Lt(key string, value any) Filter {
	return f.set(lt, key, value)
}

func (f Filter) In(key string, value any) Filter {
	return f.set(in, key, value)
}
