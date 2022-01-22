//

package dbclient

// DBConfig is used to get db instance.
type DBConfig struct {
	DBType string `toml:"db_type"`
	DBHost string `toml:"db_host"`
	DBPort int    `toml:"db_port"`
	DBPass string `toml:"db_pass"`
	DBUser string `toml:"db_user"`
	DBName string `toml:"db_name"`
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
	MaxOpenConn: 1000,
}
