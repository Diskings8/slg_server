package common_globals

func GetEnvPath() string {
	switch *CommonGlobalVarEnv {
	case "dev":
		// 根目录相对：服务统一从项目根运行（go run services/...），game_conf 的 config_path 同为根相对
		return "api/env_conf/slg.dev.yaml"
	}
	return ""
}

func IsDev() bool {
	return *CommonGlobalVarEnv == "dev"
}

func IsProd() bool {
	return *CommonGlobalVarEnv == "prod"
}

func IsTest() bool {
	return *CommonGlobalVarEnv == "test"
}
