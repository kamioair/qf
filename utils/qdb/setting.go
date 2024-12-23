package qdb

import (
	"github.com/kamioair/qf/utils/qconfig"
)

type setting struct {
	Connect string
	Config  config
}

type config struct {
	OpenLog                bool
	SkipDefaultTransaction bool
	NoLowerCase            bool
}

func loadSetting(module string) setting {
	def := setting{
		Connect: qconfig.Get(module, "db.connect", "sqlite|./db/data.db&OFF"),
		Config: config{
			OpenLog:                qconfig.Get(module, "db.config.openLog", false),
			SkipDefaultTransaction: qconfig.Get(module, "db.config.skipDefaultTransaction", true),
			NoLowerCase:            qconfig.Get(module, "db.config.noLowerCase", false),
		},
	}
	return def
}
