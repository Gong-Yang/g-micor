package app

var Conf *Config

type Config struct {
	App     AppConfig
	Mongo   MongoConfig
	Redis   RedisConfig
	Mail    MailConfig
	Observe OpenObserveConfig
	AI      AIConfig
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

type AIConfig struct {
	ApiKey  string `yaml:"apiKey"`
	BaseUrl string `yaml:"baseUrl"`
}
