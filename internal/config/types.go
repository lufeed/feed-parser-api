package config

type AppConfig struct {
	Service  ServiceConfig  `mapstructure:"service" json:"service" yaml:"service"`
	Server   ServerConfig   `mapstructure:"server" json:"server" yaml:"server"`
	Database DatabaseConfig `mapstructure:"database" json:"database" yaml:"database"`
	Cache    CacheConfig    `mapstructure:"cache" json:"cache" yaml:"cache"`
	Log      LogConfig      `mapstructure:"log" json:"log" yaml:"log"`
	Auth     AuthConfig     `mapstructure:"auth" json:"auth" yaml:"auth"`
	Proxy    ProxyConfig    `mapstructure:"proxy" json:"proxy" yaml:"proxy"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name" json:"name" yaml:"name"`
	Environment string `mapstructure:"environment" json:"environment" yaml:"environment"`
}

type ServerConfig struct {
	Host     string `mapstructure:"host" json:"host" yaml:"host"`
	Port     string `mapstructure:"port" json:"port" yaml:"port"`
	RootPath string `mapstructure:"root_path" json:"root_path" yaml:"root_path"`
}

type DatabaseConfig struct {
	Type          string `mapstructure:"type" json:"type" yaml:"type"`
	Host          string `mapstructure:"host" json:"host" yaml:"host"`
	Port          string `mapstructure:"port" json:"port" yaml:"port"`
	User          string `mapstructure:"user" json:"user" yaml:"user"`
	Pass          string `mapstructure:"pass" json:"pass" yaml:"pass"`
	Dbname        string `mapstructure:"dbname" json:"dbname" yaml:"dbname"`
	Schema        string `mapstructure:"schema" json:"schema" yaml:"schema"`
	EncryptionKey string `mapstructure:"encryption_key" json:"encryption_key" yaml:"encryption_key"`
}

type CacheConfig struct {
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	Pass    string `mapstructure:"pass" json:"pass" yaml:"pass"`
}

type LogConfig struct {
	Level  string `mapstructure:"level" json:"level" yaml:"level"`
	Format string `mapstructure:"format" json:"format" yaml:"format"`
}

type AuthConfig struct {
	APIKeys []string `mapstructure:"api_keys" json:"api_keys" yaml:"api_keys"`
}

type ProxyConfig struct {
	Proxies []Proxy `mapstructure:"proxies" json:"proxies" yaml:"proxies"`
}

type Proxy struct {
	ID       int    `mapstructure:"id" json:"id" yaml:"id"`
	Address  string `mapstructure:"address" json:"address" yaml:"address"`
	Port     string `mapstructure:"port" json:"port" yaml:"port"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}
