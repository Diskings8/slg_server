package common_globals

func GetEnvPath() string {
	switch *CommonGlobalVarEnv {
	case "env":
		return "../api/yaml_conf/slg.dev.yaml"
	}
	return ""
}
