package store

import (
	"strings"

	"github.com/go-sql-driver/mysql"
)

func FormatDSN(user, password, address, dbname, params string) string {
	paramsMap := map[string]string{
		"parseTime": "true",
	}
	for _, param := range strings.Split(params, "&") {
		split := strings.SplitN(param, "=", 2)
		if len(split) > 1 {
			paramsMap[split[0]] = split[1]
		}
	}
	mysqlConfig := &mysql.Config{
		User:   user,
		Passwd: password,
		Net:    "tcp",
		Addr:   address,
		DBName: dbname,
		Params: paramsMap,
	}
	return mysqlConfig.FormatDSN()
}
