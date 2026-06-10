package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/queue"
	"github.com/kshmirko/lidar-platform-go/internal/utils/mapper"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

var ErrTitleRequired = errors.New("title is required")

type ExperimentController struct {
	Log                           *logrus.Logger
	CreateExperimentUC            usecase.CreateExperimentUseCase
	GetExperimentByIDUC           usecase.GetExperimentByIDUseCase
	GetAllExperimentsUC           usecase.GetAllExperimentsUseCase
	GetExperimentChannelsUC       usecase.GetExperimentChannelsUseCase
	PrepareExperimentUC           usecase.PrepareExperimentUseCase
	VisualizePreparedExperimentUC usecase.VisualizePreparedExperimentUseCase
	GluePreparedExperimentUC      usecase.GluePreparedExperimentUseCase
	TaskStore                     *queue.TaskStore
}

func NewExperimentController(
	log *logrus.Logger,
	create usecase.CreateExperimentUseCase,
	getByID usecase.GetExperimentByIDUseCase,
	getAll usecase.GetAllExperimentsUseCase,
	getChannels usecase.GetExperimentChannelsUseCase,
	prepare usecase.PrepareExperimentUseCase,
	visualize usecase.VisualizePreparedExperimentUseCase,
	glue usecase.GluePreparedExperimentUseCase,
	taskStore *queue.TaskStore,
) *ExperimentController {
	return &ExperimentController{
		Log:                           log,
		CreateExperimentUC:            create,
		GetExperimentByIDUC:           getByID,
		GetAllExperimentsUC:           getAll,
		GetExperimentChannelsUC:       getChannels,
		PrepareExperimentUC:           prepare,
		VisualizePreparedExperimentUC: visualize,
		GluePreparedExperimentUC:      glue,
		TaskStore:                     taskStore,
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
func (ctrl *ExperimentController) Create(c *echo.Context) error {
	// Echo v5 requires explicit multipart form parsing before accessing FormFile
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	claims := middleware.GetClaims(c)

	title := c.FormValue("title")
	if title == "" {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: ErrTitleRequired.Error()})
	}
	comments := c.FormValue("comments")

	licelZip, err := c.FormFile("licelZip")
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	licelBgr, err := c.FormFile("licelBgr")
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	meteoFile, err := c.FormFile("meteoFile")
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	exp, err := ctrl.CreateExperimentUC.Execute(c.Request().Context(), claims.UserID, title, comments, licelZip, licelBgr, meteoFile)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, mapper.ToExperimentResponse(exp))
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
func (ctrl *ExperimentController) GetByID(c *echo.Context) error {
	var uri struct {
		ID uint `param:"id"`
	}
	if err := c.Bind(&uri); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	exp, err := ctrl.GetExperimentByIDUC.Execute(c.Request().Context(), uri.ID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, mapper.ToExperimentResponse(exp))
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
func (ctrl *ExperimentController) GetAll(c *echo.Context) error {
	var query dto.GetAllExperimentsQuery
	if err := c.Bind(&query); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
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

	result, err := ctrl.GetAllExperimentsUC.Execute(c.Request().Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, mapper.ToExperimentResponseList(result))
}

// GetChannels godoc
//
//	@Summary		Get experiment channels
//	@Description	Returns the list of available measurement channels (wavelength, polarization, photon/analog) for an experiment.
//	@Tags			experiments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		uint	true	"Experiment ID"
//	@Success		200	{object}	dto.ExperimentChannelsResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	dto.ErrorResponse	"Not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments/{id}/channels [get]
func (ctrl *ExperimentController) GetChannels(c *echo.Context) error {
	var uri struct {
		ID uint `param:"id"`
	}
	if err := c.Bind(&uri); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	channels, err := ctrl.GetExperimentChannelsUC.Execute(c.Request().Context(), uri.ID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, mapper.ToExperimentChannelsResponse(channels))
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
func (ctrl *ExperimentController) Prepare(c *echo.Context) error {
	claims := middleware.GetClaims(c)

	idStr := c.Param("id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: fmt.Sprintf("invalid experiment id: %s", idStr)})
	}

	var body dto.PrepareExperimentBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	prep, err := ctrl.PrepareExperimentUC.Execute(
		c.Request().Context(),
		claims.UserID,
		uint(experimentID),
		body.CropAlt,
		entity.BGRType(body.BGRType),
		body.BGRAlt,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		return c.JSON(code, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, mapper.ToPreparedExperimentResponse(prep))
}

// Visualize godoc
//
//	@Summary		Visualize prepared experiment data (async)
//	@Description	Enqueues a chart generation task (heatmap or profile) and returns a task ID for polling. Poll /tasks/{taskID} for the result.
//	@Tags			experiments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		uint	true	"Prepared experiment ID"
//	@Param			wavelen		query		float64	true	"Wavelength"
//	@Param			photon		query		int		false	"Photon/analog mode: 0=analog (default), 1=photon; ignored when glued=1"	default(0)
//	@Param			polarization	query		string	false	"Polarization"	default(o)
//	@Param			action		query		string	true	"image or profile"	Enums(image, profile)
//	@Param			glued		query		int		false	"Glued mode: 0=non-glued (default), 1=glued"	Enums(0, 1)	default(0)
//	@Param			type		query		string	false	"Output type: png, svg or json"	Enums(png, svg, json)	default(png)
//	@Param			formula		query		string	false	"Signal formula: raw, rangecorr, lograngecorr"	Enums(raw, rangecorr, lograngecorr)	default(raw)
//	@Param			regenerate	query		bool	false	"Force regeneration, ignoring cache"	default(false)
//	@Success		202			{object}	dto.VisualizeChartResponse
//	@Failure		400			{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401			{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404			{object}	dto.ErrorResponse	"Not found"
//	@Failure		500			{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/prepared/{id} [get]
func (ctrl *ExperimentController) Visualize(c *echo.Context) error {
	var uri dto.VisualizePreparedExperimentURI
	if err := c.Bind(&uri); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	var query dto.VisualizePreparedExperimentQuery
	if err := c.Bind(&query); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	if err := c.Validate(&query); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	if query.Type == "" {
		query.Type = "png"
	}
	if query.Formula == "" {
		query.Formula = "raw"
	}
	if query.Polarization == "" {
		query.Polarization = "o"
	}

	taskInfo, err := ctrl.VisualizePreparedExperimentUC.Execute(
		c.Request().Context(),
		uri.ID,
		query.Wavelen,
		query.Photon,
		query.Polarization,
		query.Action,
		query.Type,
		query.Formula,
		query.Regenerate,
		query.Glued,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusAccepted, dto.VisualizeChartResponse{
		TaskID: taskInfo.TaskID,
		Status: "accepted",
	})
}

// GetTaskStatus godoc
//
//	@Summary		Poll async task status
//	@Description	Returns the current status of an async task (pending, processing, done, failed). When done, includes the presigned chart URL.
//	@Tags			tasks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			taskID	path		string	true	"Task ID returned by /prepared/{id}"
//	@Success		200		{object}	dto.TaskStatusResponse
//	@Failure		404		{object}	dto.ErrorResponse	"Task not found"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/tasks/{taskID} [get]
func (ctrl *ExperimentController) GetTaskStatus(c *echo.Context) error {
	taskID := c.Param("taskID")
	if taskID == "" {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "taskID is required"})
	}

	res, err := ctrl.TaskStore.Get(c.Request().Context(), taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: fmt.Sprintf("task %s not found", taskID)})
	}

	return c.JSON(http.StatusOK, dto.TaskStatusResponse{
		TaskID: taskID,
		Status: res.Status,
		URL:    res.URL,
		Error:  res.Error,
	})
}

// Glue godoc
//
//	@Summary		Glue experiment channels
//	@Description	Starts asynchronous channel gluing for specified wavelengths and altitude range.
//	@Tags			experiments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint	true	"Experiment ID"
//	@Param			body	body		dto.GlueExperimentBody	true	"Glue parameters: wavelengths, polarization, altitude range h1-h2"
//	@Success		202		{object}	dto.MessageResponse	"Glue task submitted"
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		409		{object}	dto.ErrorResponse	"Invalid experiment status"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments/{id}/glue [post]
func (ctrl *ExperimentController) Glue(c *echo.Context) error {
	idStr := c.Param("id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: fmt.Sprintf("invalid experiment id: %s", idStr)})
	}

	var body dto.GlueExperimentBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}
	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	if err := ctrl.GluePreparedExperimentUC.Execute(
		c.Request().Context(),
		uint(experimentID),
		body.Wavelengths,
		body.Polarization,
		body.H1,
		body.H2,
	); err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		return c.JSON(code, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusAccepted, dto.MessageResponse{Message: "glue task submitted"})
}
