package common_globals

func GetEnvPath() string {
	switch *CommonGlobalVarEnv {
	case "dev":
		return "../../api/yaml_conf/slg.dev.yaml"
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
