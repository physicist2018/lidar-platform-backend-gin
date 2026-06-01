package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/utils/mapper"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

var ErrTitleRequired = errors.New("title is required")

type ExperimentController struct {
	Log                           *logrus.Logger
	CreateExperimentUC            usecase.CreateExperimentUseCase
	GetExperimentByIDUC           usecase.GetExperimentByIDUseCase
	GetAllExperimentsUC           usecase.GetAllExperimentsUseCase
	PrepareExperimentUC           usecase.PrepareExperimentUseCase
	VisualizePreparedExperimentUC usecase.VisualizePreparedExperimentUseCase
}

func NewExperimentController(
	log *logrus.Logger,
	create usecase.CreateExperimentUseCase,
	getByID usecase.GetExperimentByIDUseCase,
	getAll usecase.GetAllExperimentsUseCase,
	prepare usecase.PrepareExperimentUseCase,
	visualize usecase.VisualizePreparedExperimentUseCase,
) *ExperimentController {
	return &ExperimentController{
		Log:                           log,
		CreateExperimentUC:            create,
		GetExperimentByIDUC:           getByID,
		GetAllExperimentsUC:           getAll,
		PrepareExperimentUC:           prepare,
		VisualizePreparedExperimentUC: visualize,
	}
}

// Create godoc
//
//	@Summary		Create experiment
//	@Description	Uploads licel zip, bgr and meteo files, creates an experiment and starts asynchronous preprocessing.
//	@Tags			experiments
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			title		formData	string	true	"Experiment title"
//	@Param			comments	formData	string	false	"Experiment comments"
//	@Param			licelZip	formData	file	true	"Licel measurements zip archive"
//	@Param			licelBgr	formData	file	true	"Licel BGR file"
//	@Param			meteoFile	formData	file	true	"Meteo data file"
//	@Success		201	{object}	dto.ExperimentResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments [post]
func (ctrl *ExperimentController) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)

	title := c.PostForm("title")
	if title == "" {
		c.Error(ErrTitleRequired)
		return
	}
	comments := c.PostForm("comments")

	licelZip, err := c.FormFile("licelZip")
	if err != nil {
		c.Error(err)
		return
	}
	licelBgr, err := c.FormFile("licelBgr")
	if err != nil {
		c.Error(err)
		return
	}
	meteoFile, err := c.FormFile("meteoFile")
	if err != nil {
		c.Error(err)
		return
	}

	exp, err := ctrl.CreateExperimentUC.Execute(c.Request.Context(), claims.UserID, title, comments, licelZip, licelBgr, meteoFile)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, mapper.ToExperimentResponse(exp))
}

// GetByID godoc
//
//	@Summary		Get experiment by ID
//	@Description	Returns a single experiment by its database ID.
//	@Tags			experiments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		uint	true	"Experiment ID"
//	@Success		200	{object}	dto.ExperimentResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	dto.ErrorResponse	"Not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments/{id} [get]
func (ctrl *ExperimentController) GetByID(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required,min=1"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(err)
		return
	}

	exp, err := ctrl.GetExperimentByIDUC.Execute(c.Request.Context(), uri.ID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, mapper.ToExperimentResponse(exp))
}

// GetAll godoc
//
//	@Summary		List all experiments
//	@Description	Returns a paginated list of experiments with optional filtering.
//	@Tags			experiments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int		false	"Page number"	default(1)		minimum(1)
//	@Param			limit	query		int		false	"Items per page"	default(10)	minimum(1)	maximum(100)
//	@Param			sort	query		string	false	"Sort direction"	Enums(asc, desc)
//	@Param			status	query		string	false	"Filter by status"	Enums(staged, uploading, done, failed)
//	@Param			title	query		string	false	"Filter by title (case-insensitive partial match)"
//	@Success		200		{object}	dto.ExperimentPaginatedResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments [get]
func (ctrl *ExperimentController) GetAll(c *gin.Context) {
	var query dto.GetAllExperimentsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(err)
		return
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}

	filter := &entity.ExperimentFilter{
		Page:   query.Page,
		Limit:  query.Limit,
		Sort:   query.Sort,
		Status: entity.ExperimentStatus(query.Status),
		Title:  query.Title,
	}

	result, err := ctrl.GetAllExperimentsUC.Execute(c.Request.Context(), filter)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, mapper.ToExperimentResponseList(result))
}

// Prepare godoc
//
//	@Summary		Prepare experiment data
//	@Description	Starts background processing: background subtraction and cropping. Stores result in Minio.
//	@Tags			experiments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint	true	"Experiment ID"
//	@Param			body	body		dto.PrepareExperimentBody	true	"Preparation parameters"
//	@Success		201		{object}	dto.PreparedExperimentResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	dto.ErrorResponse	"Experiment not found"
//	@Failure		409		{object}	dto.ErrorResponse	"Experiment not ready"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments/{id}/prepare [post]
func (ctrl *ExperimentController) Prepare(c *gin.Context) {
	claims := middleware.GetClaims(c)

	idStr := c.Param("id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.Error(fmt.Errorf("invalid experiment id: %s", idStr))
		return
	}

	var body dto.PrepareExperimentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(err)
		return
	}

	prep, err := ctrl.PrepareExperimentUC.Execute(
		c.Request.Context(),
		claims.UserID,
		uint(experimentID),
		body.CropAlt,
		entity.BGRType(body.BGRType),
		body.BGRAlt,
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, mapper.ToPreparedExperimentResponse(prep))
}

// Visualize godoc
//
//	@Summary		Visualize prepared experiment data
//	@Description	Generates a heatmap or averaged profile from prepared experiment data. Returns SVG or Plotly JSON.
//	@Tags			experiments
//	@Produce		*/*
//	@Security		BearerAuth
//	@Param			id			path		uint	true	"Prepared experiment ID"
//	@Param			wavelen		path		float64	true	"Wavelength"
//	@Param			photon		path		bool	true	"Photon channel"
//	@Param			polarization	path		string	true	"Polarization"
//	@Param			action		path		string	true	"image or profile"	Enums(image, profile)
//	@Param			type		query		string	false	"Output type: svg, json or png"	Enums(svg, json, png)	default(svg)
//	@Param			formula		query		string	false	"Signal formula: raw, rangecorr, lograngecorr"	Enums(raw, rangecorr, lograngecorr)	default(raw)
//	@Success		200			{string}	string	"SVG, PNG image or Plotly JSON"
//	@Failure		400			{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401			{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404			{object}	dto.ErrorResponse	"Not found"
//	@Failure		500			{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/prepared/{id}/{wavelen}/{photon}/{polarization}/{action} [get]
func (ctrl *ExperimentController) Visualize(c *gin.Context) {
	var uri dto.VisualizePreparedExperimentURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.Error(err)
		return
	}

	var query dto.VisualizeTypeQuery
	_ = c.ShouldBindQuery(&query)
	if query.Type == "" {
		query.Type = "svg"
	}
	if query.Formula == "" {
		query.Formula = "raw"
	}

	result, err := ctrl.VisualizePreparedExperimentUC.Execute(
		c.Request.Context(),
		uri.ID,
		uri.Wavelen,
		uri.Photon,
		uri.Polarization,
		uri.Action,
		query.Type,
		query.Formula,
	)
	if err != nil {
		c.Error(err)
		return
	}

	c.Data(http.StatusOK, result.ContentType, result.Body)
}
