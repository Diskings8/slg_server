package var_globals_common

var (
	CommonGlobalVarEnv      *string
	CommonGlobalVarInstance *string
)

func init() {
	CommonGlobalVarEnv = new(string)
	CommonGlobalVarInstance = new(string)
}
