package common_globals

func GetEnvPath() string {
	switch *CommonGlobalVarEnv {
	case "dev":
		return "../../api/yaml_conf/slg.dev.yaml"
	}
	return ""
}
