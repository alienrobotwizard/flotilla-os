package app

import (
	"github.com/alienrobotwizard/flotilla-os/core/app/services"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
)

func Initialize(
	templateService services.TemplateService,
	executionService services.ExecutionService,
	workerService services.WorkerService,
) *gin.Engine {
	d := gin.Default()

	// TODO - use config here to set origins, blanket is dangerous
	d.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "PUT", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r := d.Group("/api")

	r.PUT("/template/:template_id/execute", func(c *gin.Context) {
		var request services.ExecutionRequest
		if err := c.BindJSON(&request); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		templateID := c.Param("template_id")
		request.TemplateID = &templateID
		runTemplateOrAbort(c, executionService, &request)
	})

	r.PUT("/template/name/:template_name/version/:template_version/execute", func(c *gin.Context) {
		var request services.ExecutionRequest
		if err := c.BindJSON(&request); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		templateName, templateVersion := c.Param("template_name"), c.Param("template_version")
		request.TemplateName = &templateName

		version, err := strconv.Atoi(templateVersion)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		request.SetTemplateVersion(int64(version))
		runTemplateOrAbort(c, executionService, &request)
	})

	r.GET("/template", func(c *gin.Context) {
		var request state.ListArgs

		if err := c.BindQuery(&request); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		for k, v := range c.Request.URL.Query() {
			switch k {
			case "limit", "offset", "sort_by", "order":
				continue
			default:
				request.AddFilter(k, v[0])
			}
		}

		if response, err := templateService.ListTemplates(c, &request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, response)
		}
	})

	r.POST("/template", func(c *gin.Context) {
		var template models.Template
		if err := c.BindJSON(&template); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		created, isNew, err := templateService.CreateTemplate(c, template)
		if err != nil {
			abortNotFoundOrError(c, err)
		}

		code := http.StatusCreated
		if !isNew {
			code = http.StatusOK
		}
		c.JSON(code, gin.H{
			"created":  isNew,
			"template": created,
		})
	})

	r.GET("/template/:template_id", func(c *gin.Context) {
		templateID := c.Param("template_id")
		template, err := templateService.GetTemplate(c, &state.GetTemplateArgs{TemplateID: &templateID})
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, template)
	})

	r.GET("/template/history/:run_id", func(c *gin.Context) {
		getRun(c, executionService)
	})

	r.GET("/history/:run_id", func(c *gin.Context) {
		getRun(c, executionService)
	})

	r.GET("/history", func(c *gin.Context) {
		var request state.ListRunsArgs
		if err := c.BindQuery(&request); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		for k, v := range c.Request.URL.Query() {
			switch k {
			case "limit", "offset", "sort_by", "order":
				continue
			default:
				request.AddFilter(k, v[0])
			}
		}

		if response, err := executionService.ListRuns(c, &request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, response)
		}
	})

	r.GET("/template/:template_id/history", func(c *gin.Context) {
		templateID := c.Param("template_id")
		var request state.ListRunsArgs
		request.AddFilter("template_id", templateID)

		if err := c.BindQuery(&request); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		for k, v := range c.Request.URL.Query() {
			switch k {
			case "limit", "offset", "sort_by", "order":
				continue
			default:
				request.AddFilter(k, v[0])
			}
		}

		if response, err := executionService.ListRuns(c, &request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, response)
		}
	})

	r.GET("/template/:template_id/history/:run_id", func(c *gin.Context) {
		runID := c.Param("run_id")
		run, err := executionService.GetRun(c, runID)
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, run)
	})

	r.DELETE("/template/:template_id/history/:run_id", func(c *gin.Context) {
		runID := c.Param("run_id")
		if err := executionService.Terminate(c, runID); err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, gin.H{"terminated": true})
	})

	r.GET("/history/:run_id/logs", func(c *gin.Context) {
		var lastSeen *string

		runID := c.Param("run_id")
		if val, ok := c.GetQuery("last_seen"); ok {
			lastSeen = &val
		}

		logLines, latest, err := executionService.Logs(c, runID, lastSeen)
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, gin.H{
			"log":       logLines,
			"last_seen": latest,
		})
	})

	r.GET("/worker/:engine_name", func(c *gin.Context) {
		l, err := workerService.List(c, c.Param("engine_name"))
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, l)
	})

	r.PUT("/worker", func(c *gin.Context) {
		var updates []models.Worker
		if err := c.BindJSON(&updates); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		l, err := workerService.BatchUpdate(c, updates)
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, l)
	})

	r.GET("/worker/:engine_name/:worker_type", func(c *gin.Context) {
		w, err := workerService.Get(c, c.Param("worker_type"), c.Param("engine_name"))
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, w)
	})

	r.PUT("/worker/:engine_name/:worker_type", func(c *gin.Context) {
		var updates models.Worker
		if err := c.BindJSON(&updates); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		updates.WorkerType = c.Param("worker_type")
		updates.Engine = c.Param("engine_name")
		w, err := workerService.Update(c, updates.WorkerType, updates)
		if err != nil {
			abortNotFoundOrError(c, err)
		}
		c.JSON(http.StatusOK, w)
	})

	return d
}

func getRun(c *gin.Context, executionService services.ExecutionService) {
	runID := c.Param("run_id")
	run, err := executionService.GetRun(c, runID)
	if err != nil {
		abortNotFoundOrError(c, err)
	}
	c.JSON(http.StatusOK, run)
}

func runTemplateOrAbort(c *gin.Context, service services.ExecutionService, request *services.ExecutionRequest) {
	if run, err := service.CreateTemplateRun(c, request); err != nil {
		if errors.Is(err, exceptions.ErrRecordNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		}
	} else {
		c.JSON(http.StatusCreated, run)
	}
}

func abortNotFoundOrError(c *gin.Context, err error) {
	if errors.Is(err, exceptions.ErrRecordNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
	}
	_ = c.AbortWithError(http.StatusInternalServerError, err)
}
