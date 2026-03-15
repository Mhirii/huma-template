package dto

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

type ResponseType[T any] struct {
	Body T
}

type ListQuery struct {
	Page     int    `query:"page" json:"page" doc:"Page number, starting from 1" default:"1" minimum:"1"`
	PerPage  int    `query:"per_page" json:"per_page" doc:"Number of items per page" default:"10" minimum:"1" maximum:"200"`
	SortBy   string `query:"sort_by" json:"sort_by" doc:"Sort by field" default:"created_at"`
	SortDir  string `query:"sort_dir" json:"sort_dir" doc:"Sort direction, either 'asc' or 'desc'" enum:"asc,desc" default:"desc"`
	Filters  string `query:"filters" json:"filters" doc:"Filters in JSON" default:"[]"`
	Search   string `query:"search" json:"search" doc:"Search query" default:""`
	Includes string `query:"includes" json:"includes" doc:"Includes in JSON" default:"{}"`
}

type ListQueryRes struct {
	Page     int      ` json:"page" doc:"Page number, starting from 1" default:"1" minimum:"1"`
	PerPage  int      ` json:"per_page" doc:"Number of items per page" default:"10" minimum:"1" maximum:"200"`
	SortBy   string   ` json:"sort_by" doc:"Sort by field" default:"created_at"`
	SortDir  string   ` json:"sort_dir" doc:"Sort direction, either 'asc' or 'desc'" enum:"asc,desc" default:"desc"`
	Filters  []Filter ` json:"filters" doc:"Filters in JSON" default:"[]"`
	Search   string   ` json:"search" doc:"Search query" default:""`
	Includes string   ` json:"includes" doc:"Includes in JSON" default:"{}"`
}

type AuthHeader struct {
	Authorization string `header:"Authorization" doc:"Bearer Token of the user" required:"true"`
}

type Filter struct {
	Field string `json:"field" doc:"Field to filter by"`
	Value string `json:"value" doc:"Value to filter by"`
	Rule  string `json:"rule" doc:"Rule to filter by" enum:"eq,ne,gt,gte,lt,lte,contains,ncontains,in,nin,is,nis,between,nbetween,null,nnull,empty,nempty"`
}

func ParseFilters(filters string) ([]Filter, error) {
	f := []Filter{}
	err := json.Unmarshal([]byte(filters), &f)
	return f, err
}

func ApplyFilters(filters []Filter, q *bun.SelectQuery) *bun.SelectQuery {
	for _, f := range filters {
		log.Debug().Msg(fmt.Sprintf("%s %s %s", f.Field, f.Rule, f.Value))
		// TODO: dates must be provided in UTC string format, using UNIX will cause DB exceptions
		// TODO: smartly convert dates to UTC
		field := f.Field
		value := f.Value
		switch f.Rule {
		case "contains":
			query := fmt.Sprintf("%s ILIKE ?", field)
			queryVal := fmt.Sprintf("%%%s%%", value)
			// TODO: implement inclusive/exclusive filters, WhereOr/Where
			q = q.WhereOr(query, queryVal)
		case "eq":
			q = q.Where(fmt.Sprintf("%s = ?", field), value)
		case "ne":
			q = q.Where(fmt.Sprintf("%s != ?", field), value)
		case "gt":
			q = q.Where(fmt.Sprintf("%s > ?", field), value)
		case "gte":
			q = q.Where(fmt.Sprintf("%s >= ?", field), value)
		case "lt":
			q = q.Where(fmt.Sprintf("%s < ?", field), value)
		case "lte":
			q = q.Where(fmt.Sprintf("%s <= ?", field), value)
		case "in":
			q = q.Where(fmt.Sprintf("%s IN ?", field), value)
		case "nin":
			q = q.Where(fmt.Sprintf("%s NOT IN ?", field), value)
		case "is":
			q = q.Where(fmt.Sprintf("%s IS ?", field), value)
		case "nis":
			q = q.Where(fmt.Sprintf("%s IS NOT ?", field), value)
		case "null":
			q = q.Where(fmt.Sprintf("%s IS NULL", field))
		case "nnull":
			q = q.Where(fmt.Sprintf("%s IS NOT NULL", field))
		default:
			log.Warn().Msg(fmt.Sprintf("unknown filter rule: %s", f.Rule))
		}
	}
	return q
}
