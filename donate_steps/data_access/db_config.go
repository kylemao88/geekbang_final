package data_access

type DBConfig struct {
	DbHost string `toml:"db_host"`
	DbPort int    `toml:"db_port"`
	DbPass string `toml:"db_pass"`
	DbUser string `toml:"db_user"`
	DbName string `toml:"db_name"`
}
