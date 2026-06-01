package entity

type ExperimentFilter struct {
	Page   int
	Limit  int
	Sort   string
	Status ExperimentStatus
	Title  string
	UserID uint
}
