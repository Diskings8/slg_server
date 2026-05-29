package main

import (
	"flag"

	"server.slg.com/common/configs"
	"server.slg.com/common/globals/var_globals_common"
	"server.slg.com/common/loggers"
)

const ()

var ()

func parseFlagVar() {
	flag.StringVar(&var_globals_common.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
}

func main() {
	parseFlagVar()
	flag.Parse()

	configs.LoadEnvConf(var_globals_common.GetEnvPath())

	loggers.Init()
	loggers.Log.Info(fmt.Sprintf("网关启动"))

	// pprof

	//

}
