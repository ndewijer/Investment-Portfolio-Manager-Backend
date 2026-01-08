package model

// Portfolio represents a portfolio from the database
type Portfolio struct {
	ID                  string
	Name                string
	Description         string
	IsArchived          bool
	ExcludeFromOverview bool
}

// PortfolioFilter for querying portfolios
type PortfolioFilter struct {
	IncludeArchived bool
	IncludeExcluded bool
}
