
package data_access

import (
	"database/sql"
	"fmt"
	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-sql-driver/mysql"
	"reflect"
	"strconv"
	"strings"
)

/*
var db *sql.DB

// 不支持事务
dbClient := DbClient{dbHandler:db}

// 支持事务
tx, err := db.Begin()
defer tx.Rollback()
dbClient := DbClient{dbHandler:db, tx:true}
*/

type DbClient struct {
	dbHandler interface{}
	tx        bool // 是否为事务
}

func (client *DbClient) QueryDB(sqlStr string, args []interface{}) ([]map[string]string, error) {
	debugSql := strings.Replace(sqlStr, "?", "%v", -1)
	debugSql = fmt.Sprintf(debugSql, args...)
	log.Debug("query sql = %s", debugSql)

	var rows *sql.Rows
	var err error
	if client.tx {
		txDbHandler := client.dbHandler.(*sql.Tx)
		rows, err = txDbHandler.Query(sqlStr, args...)
		if err != nil {
			log.Error("tx query db error, err = %v, args = %v, sql = %s", err, args, debugSql)
			return nil, err
		}
	} else {
		tmpDbHandler := client.dbHandler.(*sql.DB)
		rows, err = tmpDbHandler.Query(sqlStr, args...)
		if err != nil {
			log.Error("tx query db error, err = %v, args = %v, sql = %s", err, args, debugSql)
			return nil, err
		}
	}
	return client.ParseRows(rows)
}

func (client *DbClient) ParseRows(rows *sql.Rows) ([]map[string]string, error) {
	defer rows.Close()
	columns, _ := rows.Columns()
	columnsType, _ := rows.ColumnTypes()
	arr := make([]interface{}, len(columns))
	for i, v := range columnsType {
		t := v.ScanType()
		v := reflect.New(t).Interface()
		arr[i] = v
	}

	fullValues := make([]map[string]string, 0)
	for rows.Next() {
		err := rows.Scan(arr...)
		if err != nil {
			log.Error("row scan error, err = %v", err)
			return nil, err
		}
		record := make(map[string]string)
		for i, col := range columns {
			v := arr[i]
			switch vv := v.(type) {
			case *int32:
				record[col] = strconv.FormatInt(int64(*vv), 10)
			case *uint32:
				record[col] = strconv.FormatUint(uint64(*vv), 10)
			case *int64:
				record[col] = strconv.FormatInt(int64(*vv), 10)
			case *uint64:
				record[col] = strconv.FormatUint(uint64(*vv), 10)
			case *sql.NullString:
				if vv.Valid {
					record[col] = vv.String
				} else {
					record[col] = ""
				}
			case *sql.NullBool:
				if vv.Valid {
					if vv.Bool {
						record[col] = "true"
					} else {
						record[col] = "false"
					}
				} else {
					record[col] = "false"
				}
			case *sql.NullFloat64:
				if vv.Valid {
					record[col] = strconv.FormatFloat(vv.Float64, 'E', -1, 64)
				} else {
					record[col] = "0.0"
				}
			case *sql.NullInt64:
				if vv.Valid {
					record[col] = strconv.FormatInt(vv.Int64, 10)
				} else {
					record[col] = "0"
				}
			case *mysql.NullTime:
				if vv.Valid {
					record[col] = vv.Time.Format("2006-01-02 15:04:05")
				} else {
					record[col] = "2006-01-02 15:04:05"
				}
			case *sql.RawBytes:
				record[col] = string(*vv)
			default:
				record[col] = string(v.([]byte))
			}
		}
		fullValues = append(fullValues, record)
	}
	return fullValues, nil
}

func (client *DbClient) ExecDB(sqlStr string, args []interface{}) (int64, error) {
	debugSql := strings.Replace(sqlStr, "?", "%v", -1)
	debugSql = fmt.Sprintf(debugSql, args...)
	log.Debug("exec sql = %s", debugSql)

	var rowSize int64
	if client.tx {
		txDbHandler := client.dbHandler.(*sql.Tx)
		result, err := txDbHandler.Exec(sqlStr, args...)
		if err != nil {
			log.Error("exec error, err = %v, sql = %s, args = %v", err, sqlStr, args)
			return 0, err
		}
		rowSize, err = result.RowsAffected()
	} else {
		tmpDbHandler := client.dbHandler.(*sql.DB)
		result, err := tmpDbHandler.Exec(sqlStr, args...)
		if err != nil {
			log.Error("exec error, err = %v, sql = %s, args = %v", err, sqlStr, args)
			return 0, err
		}
		rowSize, err = result.RowsAffected()
	}
	log.Info("affected row size = %d", rowSize)
	return rowSize, nil
}

func (client *DbClient) InsertDataToDB(table string, data interface{}) (int64, error) {
	sqlStr := fmt.Sprintf("insert into %s (", table)
	args := make([]interface{}, 0)
	typeFields := reflect.TypeOf(data)
	valueFields := reflect.ValueOf(data)
	for i := 0; i < typeFields.NumField(); i++ {
		field := typeFields.Field(i)
		sqlStr += field.Tag.Get("json") + ", "
		value := valueFields.Field(i).Interface()
		args = append(args, value)
	}
	sqlStr = strings.TrimRight(sqlStr, ", ")
	sqlStr += ") values ("
	size := len(args)
	for i := 0; i < size; i++ {
		sqlStr += "?, "
	}
	sqlStr = strings.TrimRight(sqlStr, ", ")
	sqlStr += ")"

	rowSize, err := client.ExecDB(sqlStr, args)
	if err != nil {
		log.Error("ExecDb error, err = %v", err)
		return 0, err
	}
	return rowSize, nil
}
