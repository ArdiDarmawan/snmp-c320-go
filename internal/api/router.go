package api

import (
    "github.com/gin-gonic/gin"
    "zte-c320-snmp-api/internal/cfg"
)

func NewRouter(loader *cfg.Loader) *gin.Engine {
    r := gin.Default()

    h := &Handlers{Loader: loader}

    v1 := r.Group("/v1")
    {
        v1.GET("/health", h.Health)

        v1.GET("/olts", h.GetOlts)
        v1.GET("/olt/:name/system", h.GetSystem)
        v1.GET("/olt/:name/system/health", h.GetSystemHealth)

        v1.GET("/olt/:name/board/:slot/pon/:pon/onu", h.ListOnusByPon)              // ?detail=1
        v1.GET("/olt/:name/board/:slot/pon/:pon/onu/:onuId", h.GetOnuDetail)
        v1.GET("/olt/:name/pons", h.ListPons)
    }

    return r
}
