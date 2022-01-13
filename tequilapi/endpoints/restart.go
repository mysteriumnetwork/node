package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func AddRoutesForRestart() func(g *gin.Engine, restart func()) error {
	return func(g *gin.Engine, restart func()) error {
		g.POST("/restart", restartEndpoint(restart))
		return nil
	}
}

func restartEndpoint(restart func()) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().Msg("restart requested")
		c.AbortWithStatus(200)
		go restart()
	}
}
