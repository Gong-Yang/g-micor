package pgsql

import (
	"context"
	"time"

	"github.com/Gong-Yang/g-micor/syncx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	contextPoolKey = "pgsql_pool_key"
)

var PoolManager *poolManager

func Init(configconnString string) (err error) {
	PoolManager = &poolManager{
		store: syncx.NewResourceManager[*pgxpool.Pool](),
	}
	pool, err := PoolManager.newPool(configconnString)
	if err != nil {
		return err
	}
	PoolManager.defaultPool = pool
	PoolManager.store.Inject("", pool)
	return nil
}

type poolManager struct {
	defaultPool *pgxpool.Pool
	store       *syncx.ResourceManager[*pgxpool.Pool]
}

func (p *poolManager) Get(ctx context.Context) (res *pgxpool.Pool, err error) {
	key := ""
	poolKeyI := ctx.Value(contextPoolKey)
	if poolKeyI != nil {
		key = poolKeyI.(string)
	}
	return p.store.GetResource(key, func() (*pgxpool.Pool, error) {
		if key == "" {
			panic("please init")
		}
		//TODO
		panic("not implemented")
	})
}
func (p *poolManager) newPool(configconnString string) (*pgxpool.Pool, error) {
	// 解析配置
	config, err := pgxpool.ParseConfig(configconnString)
	if err != nil {
		return nil, err
	}
	// 配置连接池参数
	config.MaxConns = 20                      // 最大连接数
	config.MinConns = 4                       // 最小保持连接数
	config.MaxConnLifetime = 30 * time.Minute // 连接最大存活时间
	config.MaxConnIdleTime = 30 * time.Second // 空闲连接超时
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
