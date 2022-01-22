//

package dbclient

// DBConfig is used to get db instance.
type DBConfig struct {
	DBType string `yaml:"db_type"`
	DBHost string `yaml:"db_host"`
	DBPort int    `yaml:"db_port"`
	DBPass string `yaml:"db_pass"`
	DBUser string `yaml:"db_user"`
	DBName string `yaml:"db_name"`
}

// DBOptions include db option config
type DBOptions struct {
	MaxLiftTime int // minute
	MaxIdleConn int
	MaxOpenConn int
}

var defaultDBOptions = &DBOptions{
	MaxLiftTime: 15,
	MaxIdleConn: 1000,
	MaxOpenConn: 3000,
}
