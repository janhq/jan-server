package httpserver

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ServeCombinedSwagger serves the combined swagger JSON if it exists, otherwise falls back to regular swagger
func ServeCombinedSwagger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if combined swagger exists
		combinedPath := filepath.Join(".", "docs", "swagger", "swagger-combined.json")

		if _, err := os.Stat(combinedPath); err == nil {
			// Combined swagger exists, serve it
			data, err := ioutil.ReadFile(combinedPath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read combined swagger")
				c.JSON(500, gin.H{"error": "Failed to load API documentation"})
				return
			}

			var spec map[string]interface{}
			if err := json.Unmarshal(data, &spec); err != nil {
				log.Error().Err(err).Msg("Failed to parse combined swagger")
				c.JSON(500, gin.H{"error": "Failed to parse API documentation"})
				return
			}

			c.JSON(200, spec)
		} else {
			// Fall back to regular doc.json
			docPath := filepath.Join(".", "docs", "swagger", "swagger.json")
			data, err := ioutil.ReadFile(docPath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read swagger")
				c.JSON(500, gin.H{"error": "Failed to load API documentation"})
				return
			}

			var spec map[string]interface{}
			if err := json.Unmarshal(data, &spec); err != nil {
				log.Error().Err(err).Msg("Failed to parse swagger")
				c.JSON(500, gin.H{"error": "Failed to parse API documentation"})
				return
			}

			c.JSON(200, spec)
		}
	}
}
