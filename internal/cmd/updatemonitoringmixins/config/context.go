package config

type AppContext struct {
	ChartRootDir string
	DebugMode    bool
}

var appCtx *AppContext

func SetContext(ctx *AppContext) {
	appCtx = ctx
}

func GetContext() *AppContext {
	if appCtx == nil {
		panic("AppContext not initialized")
	}
	return appCtx
}
