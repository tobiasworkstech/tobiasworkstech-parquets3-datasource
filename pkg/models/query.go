package models

type QueryModel struct {
	Path              string   `json:"path"`
	Paths             []string `json:"paths"`
	PathPattern       string   `json:"pathPattern"`
	DateFormat        string   `json:"dateFormat"`
	QueryType         string   `json:"queryType"`
	VariableQueryType string   `json:"variableQueryType"`
	Prefix            string   `json:"prefix"`
	FilePattern       string   `json:"filePattern"`
	SQLQuery          string   `json:"sqlQuery"`
}
