package models

type QueryModel struct {
	Path              string `json:"path"`
	QueryType         string `json:"queryType"`
	VariableQueryType string `json:"variableQueryType"`
	Prefix            string `json:"prefix"`
	FilePattern       string `json:"filePattern"`
	SQLQuery          string `json:"sqlQuery"`
}
