package swagger

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Register exposes the swagger assets directory under /v1/swagger.
func Register(router *gin.Engine) {
	assetsDir := os.Getenv("SWAGGER_ASSETS_DIR")
	if assetsDir == "" {
		assetsDir = filepath.Join(".", "docs", "openapi")
	}
	router.StaticFS("/v1/swagger", gin.Dir(assetsDir, false))
}
