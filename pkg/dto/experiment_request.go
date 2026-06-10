package dto

type CreateExperimentBody struct {
	Title    string `form:"title"    validate:"required"`
	Comments string `form:"comments"`
}

type GetAllExperimentsQuery struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Sort   string `query:"sort"`
	Status string `query:"status"`
	Title  string `query:"title"`
}
