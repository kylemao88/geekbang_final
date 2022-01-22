//

package dbclient

import (
	"errors"

	"git.code.oa.com/gongyi/agw/log"
)

// DBClient ...
type DBClient interface {
	Query(sql_str string, args []interface{}) ([]map[string]string, error)
	Update(table string, update_field_map map[string]interface{}, condit_field_map map[string]interface{}) error
	Insert(table string, data interface{}) error
	Delete(sql_str string, args []interface{}) error
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
		panic("Register the same type of storage repeatedly")
	}
	metaStorageFactory[factory.GetType()] = factory
}

// GetMetaStoreByCfg return a new metastore by config.
func GetDBClientByCfg(cfg DBConfig) (DBClient, error) {
	if factory, ok := metaStorageFactory[cfg.DBType]; ok {
		return factory.CreateInstanceByCfg(cfg)
	}
	return nil, errors.New("client type =[" + cfg.DBType + "] is unregistered")
}

// CloseClient close storage-client and release all resource
func CloseClient(clientType string, client DBClient) {
	if client != nil {
		if factory, ok := metaStorageFactory[clientType]; ok {
			factory.DestroyInstance(client)
			return
		}
		log.Error("undefined client's type=[%s]", clientType)
	}
}
