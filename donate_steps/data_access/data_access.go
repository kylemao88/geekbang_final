// !!!!!!!!!!!!!!!!! 注意 所有设置时间不要使用 mysql的 now() 函数，全部使用go的time设置成string

package data_access

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-sql-driver/mysql"
)

var DBHandler *sql.DB

type DBProxy struct {
	DBHandler interface{}
	Tx        bool
}

func InitDBConn(db_config DBConfig) error {
	conn_str := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db_config.DbUser, db_config.DbPass, db_config.DbHost, db_config.DbPort, db_config.DbName)
	var err error
	DBHandler, err = sql.Open("mysql", conn_str)
	log.Info("db handler = %v", DBHandler)
	if err != nil {
		log.Error("open db conn error, err = %s", err.Error())
		panic("open db conn error")
	}
	// 15分钟链接没有使用就断开
	DBHandler.SetConnMaxLifetime(15 * time.Minute)
	DBHandler.SetMaxIdleConns(1000)
	DBHandler.SetMaxOpenConns(1000)
	return nil
}

func UpdateDataToTable(
	db_proxy DBProxy,
	table string,
	update_field_map map[string]interface{},
	condit_field_map map[string]interface{}) error {
	if len(table) == 0 || len(update_field_map) == 0 || len(condit_field_map) == 0 {
		return errors.New("field map empty")
	}

	sql_str := fmt.Sprintf("update %s set ", table)
	args := make([]interface{}, 0)
	for k, v := range update_field_map {
		sql_str += fmt.Sprintf("%s = ?, ", k)
		args = append(args, v)
	}
	sql_str = strings.TrimRight(sql_str, ", ")
	sql_str += " where "
	for k, v := range condit_field_map {
		sql_str += fmt.Sprintf("%s = ? and ", k)
		args = append(args, v)
	}
	sql_str = strings.TrimRight(sql_str, "and ")

	err := ExecDB(db_proxy, sql_str, args)
	if err != nil {
		log.Error("ExecDb error, err = %v", err)
		return err
	}
	return nil
}

//---------------------------------------------------------------------------------------------------------
func QueryDB(sql_str string, args []interface{}) (*sql.Rows, error) {
	debug_sql := strings.Replace(sql_str, "?", "%v", -1)
	debug_sql = fmt.Sprintf(debug_sql, args...)
	log.Debug("query sql = %s", debug_sql)

	rows, err := DBHandler.Query(sql_str, args...)
	if err != nil {
		log.Error("query db error, err = %s, sql = %s, args = %v, sql = %s",
			err.Error(), sql_str, args, debug_sql)
		return nil, err
	}

	return rows, nil
}

func TxQueryDB(db_proxy DBProxy, sql_str string, args []interface{}) (*sql.Rows, error) {
	debug_sql := strings.Replace(sql_str, "?", "%v", -1)
	debug_sql = fmt.Sprintf(debug_sql, args...)
	log.Debug("query sql = %s", debug_sql)

	if db_proxy.Tx {
		tx_db_handler := db_proxy.DBHandler.(*sql.Tx)
		rows, err := tx_db_handler.Query(sql_str, args...)
		if err != nil {
			log.Error("tx query db error, err = %s, sql = %s, args = %v, sql = %s",
				err.Error(), sql_str, args, debug_sql)
			return nil, err
		}
		return rows, nil
	} else {
		tmp_db_handler := db_proxy.DBHandler.(*sql.DB)
		rows, err := tmp_db_handler.Query(sql_str, args...)
		if err != nil {
			log.Error("tx query db error, err = %s, sql = %s, args = %v, sql = %s",
				err.Error(), sql_str, args, debug_sql)
			return nil, err
		}
		return rows, nil
	}
}

func ExecDB(db_proxy DBProxy, sql_str string, args []interface{}) error {
	debug_sql := strings.Replace(sql_str, "?", "%v", -1)
	debug_sql = fmt.Sprintf(debug_sql, args...)
	log.Debug("exec sql = %s", debug_sql)

	if db_proxy.Tx {
		tx_db_handler := db_proxy.DBHandler.(*sql.Tx)
		result, err := tx_db_handler.Exec(sql_str, args...)
		if err != nil {
			log.Error("exec error, err = %s, sql = %s, args = %v", err.Error(), sql_str, args)
			return err
		}

		row_size, err := result.RowsAffected()
		log.Info("tx affected row size = %d", row_size)
	} else {
		tmp_db_handler := db_proxy.DBHandler.(*sql.DB)
		result, err := tmp_db_handler.Exec(sql_str, args...)
		if err != nil {
			log.Error("exec error, err = %s, sql = %s, args = %v", err.Error(), sql_str, args)
			return err
		}

		row_size, err := result.RowsAffected()
		log.Info("affected row size = %d", row_size)
	}
	return nil
}

func ExecDBReturnAffected(db_proxy DBProxy, sql_str string, args []interface{}) (int64, error) {
	debug_sql := strings.Replace(sql_str, "?", "%v", -1)
	debug_sql = fmt.Sprintf(debug_sql, args...)
	log.Debug("exec sql = %s", debug_sql)

	row_size := int64(0)
	if db_proxy.Tx {
		tx_db_handler := db_proxy.DBHandler.(*sql.Tx)
		result, err := tx_db_handler.Exec(sql_str, args...)
		if err != nil {
			log.Error("exec error, err = %s, sql = %s, args = %v", err.Error(), sql_str, args)
			return row_size, err
		}

		row_size, err = result.RowsAffected()
		log.Info("tx affected row size = %d", row_size)
	} else {
		tmp_db_handler := db_proxy.DBHandler.(*sql.DB)
		result, err := tmp_db_handler.Exec(sql_str, args...)
		if err != nil {
			log.Error("exec error, err = %s, sql = %s, args = %v", err.Error(), sql_str, args)
			return row_size, err
		}

		row_size, err = result.RowsAffected()
		log.Info("affected row size = %d", row_size)
	}
	return row_size, nil
}

func InsertDataToDB(db_proxy DBProxy, table string, data interface{}) error {
	// 拼接sql
	sql_str := fmt.Sprintf("insert into %s (", table)
	args := make([]interface{}, 0)
	type_fields := reflect.TypeOf(data)
	value_fields := reflect.ValueOf(data)
	for i := 0; i < type_fields.NumField(); i++ {
		field := type_fields.Field(i)
		sql_str += field.Tag.Get("json") + ", "
		value := value_fields.Field(i).Interface()
		args = append(args, value)
	}
	sql_str = strings.TrimRight(sql_str, ", ")
	sql_str += ") values ("
	size := len(args)
	for i := 0; i < size; i++ {
		sql_str += "?, "
	}
	sql_str = strings.TrimRight(sql_str, ", ")
	sql_str += ")"
	sql_str += " ON DUPLICATE KEY UPDATE f_modify_time=f_modify_time "

	// 执行sql
	err := ExecDB(db_proxy, sql_str, args)
	if err != nil {
		log.Error("ExecDb error, err = %v", err)
		return err
	}
	return nil
}

func ParseRows(rows *sql.Rows) ([]map[string]string, error) {
	defer rows.Close()
	columns, _ := rows.Columns()
	columns_type, _ := rows.ColumnTypes()
	arr := make([]interface{}, len(columns))
	for i, v := range columns_type {
		t := v.ScanType()
		v := reflect.New(t).Interface()
		arr[i] = v
	}

	full_values := make([]map[string]string, 0)
	for rows.Next() {
		err := rows.Scan(arr...)
		if err != nil {
			log.Error("row scan error, err = %v", err)
			return nil, err
		}
		recold := make(map[string]string)
		for i, col := range columns {
			v := arr[i]
			switch vv := v.(type) {
			case *int32:
				recold[col] = strconv.FormatInt(int64(*vv), 10)
			case *uint32:
				recold[col] = strconv.FormatUint(uint64(*vv), 10)
			case *int64:
				recold[col] = strconv.FormatInt(int64(*vv), 10)
			case *uint64:
				recold[col] = strconv.FormatUint(uint64(*vv), 10)
			case *sql.NullString:
				if vv.Valid {
					recold[col] = vv.String
				} else {
					recold[col] = ""
				}
			case *sql.NullBool:
				if vv.Valid {
					if vv.Bool {
						recold[col] = "true"
					} else {
						recold[col] = "false"
					}
				} else {
					recold[col] = "false"
				}
			case *sql.NullFloat64:
				if vv.Valid {
					recold[col] = strconv.FormatFloat(vv.Float64, 'E', -1, 64)
				} else {
					recold[col] = "0.0"
				}
			case *sql.NullInt64:
				if vv.Valid {
					recold[col] = strconv.FormatInt(vv.Int64, 10)
				} else {
					recold[col] = "0"
				}
			case *mysql.NullTime:
				if vv.Valid {
					recold[col] = vv.Time.Format("2006-01-02 15:04:05")
				} else {
					recold[col] = "2006-01-02 15:04:05"
				}
			case *sql.RawBytes:
				recold[col] = string(*vv)
			default:
				recold[col] = string(v.([]byte))
			}
		}
		full_values = append(full_values, recold)
	}
	return full_values, nil
}
