//

package dbclient

import (
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
)

// DBClient ...
type DBClient interface {
	Query(sql_str string, args []interface{}) ([]map[string]string, error)
	Update(table string, update_field_map map[string]interface{}, condit_field_map map[string]interface{}) error
	Insert(table string, data interface{}) error
	Delete(sql_str string, args []interface{}) error
	ExecSQL(sql_str string, args []interface{}) (int64, error)
	TxBegin() (interface{}, error)
	TxCommit(interface{}) error
	TxRollback(interface{}) error
	TxInsert(tx interface{}, table string, data interface{}) error
	TxExecSQL(tx interface{}, sql_str string, args []interface{}) (int64, error)
	Prepare(sqlStr string) (interface{}, error)
}

// ClientFactory create client instance by type
type ClientFactory interface {
	// GetType return the storage' name
	GetType() string
	// CreateInstanceByCfg create a storage instance
	CreateInstanceByCfg(cfg DBConfig) (DBClient, error)
	// DestroyInstance destroy instance
	DestroyInstance(DBClient)
}

var (
	metaStorageFactory = make(map[string]ClientFactory)
)

// RegisterClientType register a new client type
func RegisterClient(factory ClientFactory) {
	_, ok := metaStorageFactory[factory.GetType()]
	if ok {
		panic("RegisterClient - Register the same type of storage repeatedly")
	}
	metaStorageFactory[factory.GetType()] = factory
}

// GetMetaStoreByCfg return a new metastore by config.
func GetDBClientByCfg(cfg DBConfig) (DBClient, error) {
	if factory, ok := metaStorageFactory[cfg.DBType]; ok {
		return factory.CreateInstanceByCfg(cfg)
	}
	return nil, errors.NewDBClientError(errors.WithMsg("GetDBClientByCfg - client type =[%v] is unregistered", cfg.DBType))
}

// CloseClient close storage-client and release all resource
func CloseClient(clientType string, client DBClient) error {
	if client != nil {
		if factory, ok := metaStorageFactory[clientType]; ok {
			factory.DestroyInstance(client)
			return nil
		}
		return errors.NewDBClientError(errors.WithMsg("CloseClient - undefined client's type=[%s]", clientType))
	}
	return nil
}
