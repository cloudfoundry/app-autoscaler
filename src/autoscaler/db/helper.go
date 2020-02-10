package db

import (
	"fmt"
	"strings"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"github.com/go-sql-driver/mysql"

)

type Database struct {
	DriverName  string
	DSN         string
}

/**
 This function is used to generate db connection info, for example, 
 For mysql: 
 input dbUrl: 'username:password@tcp(localhost:3306)/autoscaler?tls=custom&sslrootcert=db_ca.crt'
   return:
   &Database{DriverName: "mysql", DSN:"username:password@tcp(localhost:3306)/autoscaler?parseTime=true&tls=custom"}
 
For postgres:
   input dbUrl: postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable
   return:
   &Database{DriverName: "postgres", DSN:"postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable"
 **/
 func Connection(dbUrl string) (*Database, error) {
	var dsn string
	var tlsValue string
	var sslrootcert string
	database := &Database{}
	queryString := "?parseTime=true"

	database.DriverName = detectDirver(dbUrl)

	switch database.DriverName {
	case MysqlDriverName:
		if strings.Contains(dbUrl,"?") {
			params := []string{}
			urlStrs := strings.Split(dbUrl,"?")
			paramString := urlStrs[1]
			for _, v := range strings.Split(paramString, "&") {
				param := strings.SplitN(v, "=", 2)
				if len(param) != 2 {
					continue
				}
				if param[0] == "tls" {
					tlsValue= param[1]
				}
				if param[0]=="sslrootcert" {
					sslrootcert = param[1]
					continue
				}
				params = append(params,v)
			}

			queryString = fmt.Sprintf("%s&%s", queryString, strings.Join(params,"&"))
			dsn = urlStrs[0]+queryString
			database.DSN = dsn

			err :=registerConfig(tlsValue,sslrootcert,urlStrs[0])
			if err !=nil {
				return nil, err
			}

		}else {
			database.DSN = dbUrl+queryString
		}				
	case PostgresDriverName:
		database.DSN = dbUrl
	}
	return database, nil
}

func registerConfig(tlsValue string, sslrootcert string, url string) error {
	if _, isBool := readBool(tlsValue); isBool || strings.ToLower(tlsValue) == "skip-verify" || strings.ToLower(tlsValue) == "preferred" {
		// Do nothing here
		return nil
	} else if sslrootcert != "" {
		certBytes, err := ioutil.ReadFile(sslrootcert)
		if err != nil {
			fmt.Printf("failed to read sql ca file: %v", err)
			return err
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
			fmt.Printf("failed to parse sql ca: %v", err)
			return err
		}

		cfg, err := mysql.ParseDSN(url)
		if err != nil {
			fmt.Printf("invalid db url %s with error %v",url, err)
			return err
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
			ServerName:         strings.Split(cfg.Addr,":")[0],
		}
		err = mysql.RegisterTLSConfig(tlsValue, tlsConfig)
		if err != nil {
			fmt.Printf("failed to register tlsconfig: %v",err)
			return err
		}

	} else {
		return fmt.Errorf("sql ca file is not provided when tls is a custom key")
	}
	return nil
}

func readBool(input string) (value bool, valid bool) {
	switch input {
	case "1", "true", "TRUE", "True":
		return true, true
	case "0", "false", "FALSE", "False":
		return false, true
	}
	return
}

func detectDirver(dbUrl string)(driver string) {
	if strings.Contains(dbUrl, "postgres"){
		return PostgresDriverName
	} else {
		return MysqlDriverName
	}
}

