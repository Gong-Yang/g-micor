package app

var Conf *Config

type Config struct {
	App     AppConfig
	Mongo   MongoConfig
	Redis   RedisConfig
	Mail    MailConfig
	Observe OpenObserveConfig
}

type AppConfig struct {
	Name       string
	Port       int
	CenterAddr string `yaml:"centerAddr"`
	LogLevel   string
}

type RedisConfig struct {
	Addr     string
	Password string
	Db       int
	TLS      bool
}

type MongoConfig struct {
	Uri      string
	Database string
}
type MailConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type OpenObserveConfig struct {
	Endpoint     string
	Organization string
	Stream       string
	Username     string
	Password     string
}
