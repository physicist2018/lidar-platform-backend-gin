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

	"github.com/physicist2018/lidar-platform-go/internal/delivery/http/middleware"
	"github.com/physicist2018/lidar-platform-go/internal/delivery/http/response"
	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
	"github.com/physicist2018/lidar-platform-go/internal/utils/mapper"
	"github.com/physicist2018/lidar-platform-go/pkg/dto"
)

var ErrTitleRequired = errors.New("title is required")

type ExperimentController struct {
	Log                      *logrus.Logger
	CreateExperimentUC       usecase.CreateExperimentUseCase
	GetExperimentByIDUC      usecase.GetExperimentByIDUseCase
	GetAllExperimentsUC      usecase.GetAllExperimentsUseCase
	GetExperimentChannelsUC  usecase.GetExperimentChannelsUseCase
	ProcessExperimentUC      usecase.ProcessExperimentUseCase
	GetProcessingRunStatusUC usecase.GetProcessingRunStatusUseCase
	Validate                 *validator.Validate
}

func NewExperimentController(
	log *logrus.Logger,
	create usecase.CreateExperimentUseCase,
	getByID usecase.GetExperimentByIDUseCase,
	getAll usecase.GetAllExperimentsUseCase,
	getChannels usecase.GetExperimentChannelsUseCase,
	process usecase.ProcessExperimentUseCase,
	getProcessStatus usecase.GetProcessingRunStatusUseCase,
	validate *validator.Validate,
) *ExperimentController {
	return &ExperimentController{
		Log:                      log,
		CreateExperimentUC:       create,
		GetExperimentByIDUC:      getByID,
		GetAllExperimentsUC:      getAll,
		GetExperimentChannelsUC:  getChannels,
		ProcessExperimentUC:      process,
		GetProcessingRunStatusUC: getProcessStatus,
		Validate:                 validate,
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

// Process godoc
//
//	@Summary		Run processing algorithm on experiment
//	@Description	Starts an async processing run (stage0, stage1, ...) on the experiment.
//	@Tags			experiments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint					true	"Experiment ID"
//	@Param			body	body		dto.ProcessExperimentBody	true	"Processing parameters"
//	@Success		201		{object}	dto.ProcessingRunResponse
//	@Failure		400		{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404		{object}	dto.ErrorResponse	"Experiment not found"
//	@Failure		409		{object}	dto.ErrorResponse	"Experiment not ready"
//	@Failure		500		{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/experiments/{id}/process [post]
func (ctrl *ExperimentController) Process(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)

	idStr := chi.URLParam(r, "id")
	experimentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("invalid experiment id: %s", idStr))
		return
	}

	var body dto.ProcessExperimentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctrl.Validate.Struct(&body); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	run, err := ctrl.ProcessExperimentUC.Execute(
		r.Context(),
		claims.UserID,
		uint(experimentID),
		body.Algorithm,
		body.Params,
	)
	if err != nil {
		code := http.StatusInternalServerError
		if ce, ok := err.(interface{ StatusCode() int }); ok {
			code = ce.StatusCode()
		}
		response.Error(w, code, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, mapper.ToProcessingRunResponse(run))
}

// GetProcessingStatus godoc
//
//	@Summary		Get processing run status
//	@Description	Returns the current status of a processing run.
//	@Tags			experiments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		uint	true	"Processing Run ID"
//	@Success		200	{object}	dto.ProcessingRunResponse
//	@Failure		400	{object}	dto.ErrorResponse	"Bad request"
//	@Failure		401	{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	dto.ErrorResponse	"Not found"
//	@Failure		500	{object}	dto.ErrorResponse	"Internal server error"
//	@Router			/processing/{id} [get]
func (ctrl *ExperimentController) GetProcessingStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := parseUint(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	run, err := ctrl.GetProcessingRunStatusUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, mapper.ToProcessingRunResponse(run))
}
