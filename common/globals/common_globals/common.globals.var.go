package common_globals

var (
	CommonGlobalVarEnv      *string
	CommonGlobalVarInstance *string
)

func init() {
	CommonGlobalVarEnv = new(string)
	CommonGlobalVarInstance = new(string)
}
