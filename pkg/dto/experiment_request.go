package dto

type CreateExperimentBody struct {
	Title    string `form:"title"    binding:"required"`
	Comments string `form:"comments"`
}

type GetAllExperimentsQuery struct {
	Page   int    `form:"page"  binding:"omitempty,min=1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort   string `form:"sort"  binding:"omitempty,oneof=asc desc"`
	Status string `form:"status"`
	Title  string `form:"title"`
}
