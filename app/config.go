package app

var Conf *Config

type Config struct {
	App     AppConfig
	Mongo   MongoConfig
	PGSQL   PGSQLConfig `yaml:"pgSQL"`
	Redis   RedisConfig
	Observe OpenObserveConfig
}

type AppConfig struct {
	Name       string
	Port       int
	RpcPort    int    `yaml:"rpcPort"`
	CenterAddr string `yaml:"centerAddr"`
	LogLevel   string
	HmacKey    string `yaml:"hmacKey"`
}

type RedisConfig struct {
	Addr     string
	Password string
	Db       int
}

type PGSQLConfig struct {
	Uri string
}
type MongoConfig struct {
	Uri      string
	Database string
}
type OpenObserveConfig struct {
	Endpoint     string
	Organization string
	Stream       string
	Username     string
	Password     string
}
