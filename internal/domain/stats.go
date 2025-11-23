package domain

type UserAssignmentsStat struct {
	UserID                 string
	ReviewAssignmentsCount int
}

type PRStatusCounts struct {
	Open   int
	Merged int
	Total  int
}

type Stats struct {
	AssignmentsByUser []UserAssignmentsStat
	PRStatusCounts    PRStatusCounts
}
