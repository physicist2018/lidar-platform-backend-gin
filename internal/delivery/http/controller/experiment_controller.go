package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/response"
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
	Validate                      *validator.Validate
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
	validate *validator.Validate,
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
		Validate:                      validate,
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
func (ctrl *ExperimentController) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	claims := middleware.GetClaims(r)

	title := r.FormValue("title")
	if title == "" {
		response.Error(w, http.StatusBadRequest, ErrTitleRequired.Error())
		return
	}
	comments := r.FormValue("comments")

	_, licelZip, err := r.FormFile("licelZip")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_, licelBgr, err := r.FormFile("licelBgr")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_, meteoFile, err := r.FormFile("meteoFile")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	exp, err := ctrl.CreateExperimentUC.Execute(r.Context(), claims.UserID, title, comments, licelZip, licelBgr, meteoFile)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, mapper.ToExperimentResponse(exp))
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
func (ctrl *ExperimentController) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	exp, err := ctrl.GetExperimentByIDUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, mapper.ToExperimentResponse(exp))
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
func (ctrl *ExperimentController) GetAll(w http.ResponseWriter, r *http.Request) {
	query := dto.GetAllExperimentsQuery{
		Page:  1,
		Limit: 10,
	}

	q := r.URL.Query()
	if v := q.Get("page"); v != "" {
		query.Page = parseInt(v, 1)
	}
	if v := q.Get("limit"); v != "" {
		query.Limit = parseInt(v, 10)
	}
	query.Sort = q.Get("sort")
	query.Status = q.Get("status")
	query.Title = q.Get("title")

	filter := &entity.ExperimentFilter{
		Page:   query.Page,
		Limit:  query.Limit,
		Sort:   query.Sort,
		Status: entity.ExperimentStatus(query.Status),
		Title:  query.Title,
	}

	result, err := ctrl.GetAllExperimentsUC.Execute(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, mapper.ToExperimentResponseList(result))
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
func (ctrl *ExperimentController) GetChannels(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	channels, err := ctrl.GetExperimentChannelsUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, mapper.ToExperimentChannelsResponse(channels))
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
func (ctrl *ExperimentController) Prepare(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	idStr := chi.URLParam(r, "id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("invalid experiment id: %s", idStr))
		return
	}

	var body dto.PrepareExperimentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	prep, err := ctrl.PrepareExperimentUC.Execute(
		r.Context(),
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
		response.Error(w, code, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, mapper.ToPreparedExperimentResponse(prep))
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
func (ctrl *ExperimentController) Visualize(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	q := r.URL.Query()

	action := q.Get("action")
	if action == "" {
		response.Error(w, http.StatusBadRequest, "action is required")
		return
	}

	wavelen, _ := strconv.ParseFloat(q.Get("wavelen"), 64)
	photon, _ := strconv.Atoi(q.Get("photon"))
	polarization := q.Get("polarization")
	if polarization == "" {
		polarization = "o"
	}
	outputType := q.Get("type")
	if outputType == "" {
		outputType = "png"
	}
	formula := q.Get("formula")
	if formula == "" {
		formula = "raw"
	}
	glued, _ := strconv.Atoi(q.Get("glued"))

	taskInfo, err := ctrl.VisualizePreparedExperimentUC.Execute(
		r.Context(),
		id,
		wavelen,
		int8(photon),
		polarization,
		action,
		outputType,
		formula,
		q.Get("regenerate") == "true",
		int8(glued),
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusAccepted, dto.VisualizeChartResponse{
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
func (ctrl *ExperimentController) GetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		response.Error(w, http.StatusBadRequest, "taskID is required")
		return
	}

	res, err := ctrl.TaskStore.Get(r.Context(), taskID)
	if err != nil {
		response.Error(w, http.StatusNotFound, fmt.Sprintf("task %s not found", taskID))
		return
	}

	response.JSON(w, http.StatusOK, dto.TaskStatusResponse{
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
func (ctrl *ExperimentController) Glue(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("invalid experiment id: %s", idStr))
		return
	}

	var body dto.GlueExperimentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := ctrl.GluePreparedExperimentUC.Execute(
		r.Context(),
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
		response.Error(w, code, err.Error())
		return
	}

	response.JSON(w, http.StatusAccepted, dto.MessageResponse{Message: "glue task submitted"})
}
