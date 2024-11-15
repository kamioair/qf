package qdb

import (
	"errors"
	"fmt"
	"github.com/kamioair/qf/utils/qio"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"strings"
)

var dbCaches = make(map[string]*gorm.DB)

// NewDb
//
//	@Description: 创建数据库
//	@param cfgSection 配置节点，用于启动多个数据库不用配置
//	@return *gorm.DB
func NewDb(module string) *gorm.DB {
	// 如果缓存已经存在，则直接返回
	if cdb, ok := dbCaches[module]; ok {
		return cdb
	}

	// 加载配置
	setting := loadSetting(module)
	gc := gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			NoLowerCase:   true,
		},
		SkipDefaultTransaction: setting.Config.SkipDefaultTransaction,
	}
	if setting.Config.OpenLog {
		gc.Logger = logger.Default.LogMode(logger.Info)
	}
	sp := strings.Split(setting.Connect, "|")

	// 创建数据库连接
	var db *gorm.DB
	var err error
	switch sp[0] {
	case "sqlite":
		spp := strings.Split(sp[1], "&")
		// 创建数据库
		file := qio.GetFullPath(spp[0])
		if _, err := qio.CreateDirectory(file); err != nil {
			panic(err)
		}
		db, err = gorm.Open(sqlite.Open(file), &gc)
		if err != nil {
			panic(err)
		}
		// Journal模式
		//  DELETE：在事务提交后，删除journal文件
		//  MEMORY：在内存中生成journal文件，不写入磁盘
		//  WAL：使用WAL（Write-Ahead Logging）模式，将journal记录写入WAL文件中
		//  OFF：完全关闭journal模式，不记录任何日志消息
		if spp[1] != "" {
			db.Exec(fmt.Sprintf("PRAGMA journal_mode = %s;", spp[1]))
		}
	case "sqlserver":
		dsn := fmt.Sprintf("sqlserver://%s", sp[1])
		db, err = gorm.Open(sqlserver.Open(dsn), &gc)
		if err != nil {
			panic(err)
		}
	case "mysql":
		dsn := sp[1]
		db, err = gorm.Open(mysql.Open(dsn), &gc)
		if err != nil {
			panic(err)
		}
	}
	if db == nil {
		panic(errors.New("unknown db type"))
	}
	dbCaches[module] = db
	return db
}
