package app

import (
	"github.com/alienrobotwizard/flotilla-os/core/app/services"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func Initialize(
	templateService services.TemplateService, executionService services.ExecutionService) *gin.Engine {
	d := gin.Default()
	r := d.Group("/api")

	r.PUT("/template/:template_id/execute", func(c *gin.Context) {
		var request services.ExecutionRequest
		if err := c.BindJSON(&request); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		}

		templateID := c.Param("template_id")
		request.TemplateID = &templateID

		if run, err := executionService.CreateTemplateRun(&request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusCreated, run)
		}
	})

	r.PUT("/template/name/:template_name/version/:template_version/execute", func(c *gin.Context) {
		var request services.ExecutionRequest
		if err := c.BindJSON(&request); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		}

		templateName, templateVersion := c.Param("template_name"), c.Param("template_version")
		request.TemplateName = &templateName

		if templateVersion, err := strconv.Atoi(templateVersion); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		} else {
			templateVersion := int64(templateVersion)
			request.TemplateVersion = &templateVersion
		}

		if run, err := executionService.CreateTemplateRun(&request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusCreated, run)
		}
	})

	r.GET("/template", func(c *gin.Context) {
		var request state.ListArgs

		if err := c.BindQuery(&request); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		}

		for k, v := range c.Request.URL.Query() {
			switch k {
			case "limit", "offset", "sort_by", "order":
				continue
			default:
				request.AddFilter(k, v[0])
			}
		}

		if response, err := templateService.ListTemplates(&request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, response)
		}
	})

	r.POST("/template", func(c *gin.Context) {
		var template models.Template
		if err := c.BindJSON(&template); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		}

		if created, err := templateService.CreateTemplate(template); err != nil {
			switch err.(type) {
			case exceptions.MalformedInput:
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				_ = c.AbortWithError(http.StatusInternalServerError, err)
			}
		} else {
			c.JSON(http.StatusCreated, gin.H{
				"created":  true,
				"template": created,
			})
		}
	})

	r.GET("/template/:template_id", func(c *gin.Context) {
		templateID := c.Param("template_id")
		if template, err := templateService.GetTemplate(&state.GetTemplateArgs{TemplateID: &templateID}); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, template)
		}
	})

	r.GET("/template/history/:run_id", func(c *gin.Context) {
		runID := c.Param("run_id")
		if run, err := executionService.GetRun(runID); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, run)
		}
	})

	r.GET("/template/:template_id/history", func(c *gin.Context) {
		templateID := c.Param("template_id")
		var request state.ListRunsArgs
		request.AddFilter("template_id", templateID)

		if err := c.BindQuery(&request); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
		}

		for k, v := range c.Request.URL.Query() {
			switch k {
			case "limit", "offset", "sort_by", "order":
				continue
			default:
				request.AddFilter(k, v[0])
			}
		}

		if response, err := executionService.ListRuns(&request); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, response)
		}
	})

	r.GET("/template/:template_id/history/:run_id", func(c *gin.Context) {
		runID := c.Param("run_id")
		if run, err := executionService.GetRun(runID); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, run)
		}
	})

	r.DELETE("/template/:template_id/history/:run_id", func(c *gin.Context) {
		runID := c.Param("run_id")
		if err := executionService.Terminate(runID); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, gin.H{"terminated": true})
		}
	})

	return d
}
