package duckdb

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/tobiasworkstech/parquets3-datasource/pkg/parquet"
)

// Executor handles SQL queries on parquet files
type Executor struct {
	s3Client *s3.Client
	bucket   string
}

// NewExecutor creates a new executor
func NewExecutor(s3Client *s3.Client, bucket string) *Executor {
	return &Executor{
		s3Client: s3Client,
		bucket:   bucket,
	}
}

// ExecuteSQL runs a SQL query on a parquet file from S3
func (e *Executor) ExecuteSQL(ctx context.Context, key, sqlQuery string) ([]*data.Frame, error) {
	// Read the parquet file using the existing reader
	frames, err := parquet.ReadParquetFromS3(ctx, e.s3Client, e.bucket, key)
	if err != nil {
		return nil, fmt.Errorf("read parquet file: %w", err)
	}

	if len(frames) == 0 {
		return frames, nil
	}

	frame := frames[0]

	// Parse and execute the SQL query
	result, err := executeSQLOnFrame(frame, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("execute SQL: %w", err)
	}

	// Add the SQL query as metadata on the frame
	if result.Meta == nil {
		result.Meta = &data.FrameMeta{}
	}
	result.Meta.ExecutedQueryString = sqlQuery

	return []*data.Frame{result}, nil
}

// executeSQLOnFrame parses SQL and applies it to a data frame
func executeSQLOnFrame(frame *data.Frame, sqlQuery string) (*data.Frame, error) {
	// Parse the SQL query manually (simplified parser)
	return executeSimpleSQL(frame, sqlQuery)
}

// executeSimpleSQL handles SQL patterns without full parsing
func executeSimpleSQL(frame *data.Frame, sqlQuery string) (*data.Frame, error) {
	query := strings.TrimSpace(sqlQuery)
	upperQuery := strings.ToUpper(query)

	// Check if it's a SELECT statement
	if !strings.HasPrefix(upperQuery, "SELECT") {
		return nil, fmt.Errorf("only SELECT statements are supported")
	}

	var resultFrame *data.Frame = frame
	var err error

	// Extract columns (between SELECT and FROM)
	selectIdx := 6 // After "SELECT"
	fromIdx := strings.Index(upperQuery, "FROM")
	if fromIdx == -1 {
		fromIdx = len(query)
	}

	columnsStr := strings.TrimSpace(query[selectIdx:fromIdx])

	// Check for GROUP BY clause
	groupByIdx := strings.Index(upperQuery, "GROUP BY")
	var groupByColumns []string
	if groupByIdx != -1 {
		// Find the end of GROUP BY clause
		groupByEnd := len(query)
		if idx := strings.Index(upperQuery[groupByIdx:], "ORDER BY"); idx != -1 {
			groupByEnd = groupByIdx + idx
		}
		if idx := strings.Index(upperQuery[groupByIdx:], "LIMIT"); idx != -1 && groupByIdx+idx < groupByEnd {
			groupByEnd = groupByIdx + idx
		}
		groupByStr := strings.TrimSpace(query[groupByIdx+8 : groupByEnd])
		groupByColumns = parseColumnList(groupByStr)
	}

	// Apply WHERE clause first
	whereIdx := strings.Index(upperQuery, "WHERE")
	if whereIdx != -1 {
		// Find the end of WHERE clause
		whereEnd := len(query)
		if idx := strings.Index(upperQuery[whereIdx:], "GROUP BY"); idx != -1 {
			whereEnd = whereIdx + idx
		}
		if idx := strings.Index(upperQuery[whereIdx:], "ORDER BY"); idx != -1 && whereIdx+idx < whereEnd {
			whereEnd = whereIdx + idx
		}
		if idx := strings.Index(upperQuery[whereIdx:], "LIMIT"); idx != -1 && whereIdx+idx < whereEnd {
			whereEnd = whereIdx + idx
		}

		whereClause := strings.TrimSpace(query[whereIdx+5 : whereEnd])
		resultFrame, err = applyWhereClause(resultFrame, whereClause)
		if err != nil {
			return nil, err
		}
	}

	// Check if query has aggregation functions
	hasAggregation := strings.Contains(upperQuery, "COUNT(") ||
		strings.Contains(upperQuery, "AVG(") ||
		strings.Contains(upperQuery, "SUM(") ||
		strings.Contains(upperQuery, "MIN(") ||
		strings.Contains(upperQuery, "MAX(")

	if hasAggregation || len(groupByColumns) > 0 {
		// Parse aggregation columns
		resultFrame, err = applyAggregation(resultFrame, columnsStr, groupByColumns)
		if err != nil {
			return nil, err
		}
	} else {
		// Apply column selection if not SELECT *
		if columnsStr != "*" && !strings.HasPrefix(columnsStr, "* ") {
			columns := parseColumnList(columnsStr)
			if len(columns) > 0 {
				resultFrame, err = selectColumns(frame, columns)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Apply ORDER BY clause
	orderByIdx := strings.Index(upperQuery, "ORDER BY")
	if orderByIdx != -1 {
		// Find the end of ORDER BY clause (LIMIT or end of string)
		orderByEnd := len(query)
		if idx := strings.Index(upperQuery[orderByIdx:], "LIMIT"); idx != -1 {
			orderByEnd = orderByIdx + idx
		}

		orderByClause := strings.TrimSpace(query[orderByIdx+8 : orderByEnd])
		resultFrame, err = applyOrderByClause(resultFrame, orderByClause)
		if err != nil {
			return nil, err
		}
	}

	// Apply LIMIT clause
	limitIdx := strings.Index(upperQuery, "LIMIT")
	if limitIdx != -1 {
		limitStr := strings.TrimSpace(query[limitIdx+5:])
		limitStr = strings.Split(limitStr, " ")[0]
		limit, err := strconv.Atoi(limitStr)
		if err == nil && limit > 0 {
			resultFrame = limitFrame(resultFrame, limit)
		}
	}

	return resultFrame, nil
}

// aggregationColumn represents a parsed aggregation expression
type aggregationColumn struct {
	function string // COUNT, AVG, SUM, MIN, MAX, or empty for regular column
	column   string // column name or * for COUNT(*)
	alias    string // AS alias name
}

// parseAggregationColumns parses SELECT columns including aggregation functions
func parseAggregationColumns(columnsStr string) []aggregationColumn {
	var result []aggregationColumn
	// Split by comma, but handle nested parentheses
	parts := splitByComma(columnsStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		col := aggregationColumn{}

		// Check for alias
		upperPart := strings.ToUpper(part)
		if idx := strings.Index(upperPart, " AS "); idx != -1 {
			col.alias = strings.TrimSpace(part[idx+4:])
			col.alias = strings.Trim(col.alias, `"'`)
			part = strings.TrimSpace(part[:idx])
			upperPart = strings.ToUpper(part)
		}

		// Check for aggregation functions
		for _, fn := range []string{"COUNT", "AVG", "SUM", "MIN", "MAX", "ROUND"} {
			if strings.HasPrefix(upperPart, fn+"(") {
				col.function = fn
				// Extract column from function
				start := strings.Index(part, "(")
				end := strings.LastIndex(part, ")")
				if start != -1 && end != -1 {
					inner := strings.TrimSpace(part[start+1 : end])
					// Handle ROUND(AVG(...), 2) - nested functions
					if strings.Contains(strings.ToUpper(inner), "AVG(") {
						col.function = "AVG"
						avgStart := strings.Index(strings.ToUpper(inner), "AVG(")
						avgEnd := strings.Index(inner[avgStart:], ")")
						if avgEnd != -1 {
							col.column = strings.TrimSpace(inner[avgStart+4 : avgStart+avgEnd])
						}
					} else if strings.Contains(strings.ToUpper(inner), "SUM(") {
						col.function = "SUM"
						sumStart := strings.Index(strings.ToUpper(inner), "SUM(")
						sumEnd := strings.Index(inner[sumStart:], ")")
						if sumEnd != -1 {
							col.column = strings.TrimSpace(inner[sumStart+4 : sumStart+sumEnd])
						}
					} else if strings.Contains(strings.ToUpper(inner), "COUNT(") {
						col.function = "COUNT"
						countStart := strings.Index(strings.ToUpper(inner), "COUNT(")
						countEnd := strings.Index(inner[countStart:], ")")
						if countEnd != -1 {
							col.column = strings.TrimSpace(inner[countStart+6 : countStart+countEnd])
						}
					} else {
						col.column = inner
					}
				}
				break
			}
		}

		// If no function, it's a regular column
		if col.function == "" {
			col.column = strings.Trim(part, `"'`)
		}

		col.column = strings.Trim(col.column, `"'`)
		result = append(result, col)
	}

	return result
}

// splitByComma splits a string by comma, respecting parentheses
func splitByComma(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, c := range s {
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
		} else if c == ',' && depth == 0 {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(c)
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// applyAggregation applies aggregation functions and GROUP BY
func applyAggregation(frame *data.Frame, columnsStr string, groupByColumns []string) (*data.Frame, error) {
	aggCols := parseAggregationColumns(columnsStr)
	numRows, _ := frame.RowLen()

	// If no GROUP BY, aggregate entire dataset
	if len(groupByColumns) == 0 {
		return aggregateAll(frame, aggCols, numRows)
	}

	// GROUP BY aggregation
	return aggregateByGroup(frame, aggCols, groupByColumns, numRows)
}

// aggregateAll aggregates the entire frame without grouping
func aggregateAll(frame *data.Frame, aggCols []aggregationColumn, numRows int) (*data.Frame, error) {
	newFrame := data.NewFrame(frame.Name)

	for _, col := range aggCols {
		var value interface{}
		alias := col.alias
		if alias == "" {
			if col.function != "" {
				alias = strings.ToLower(col.function) + "_" + col.column
			} else {
				alias = col.column
			}
		}

		switch col.function {
		case "COUNT":
			value = int64(numRows)
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, []int64{value.(int64)}))
		case "AVG":
			avg := calculateAvg(frame, col.column, numRows)
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, []float64{avg}))
		case "SUM":
			sum := calculateSum(frame, col.column, numRows)
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, []float64{sum}))
		case "MIN":
			min := calculateMin(frame, col.column, numRows)
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, []float64{min}))
		case "MAX":
			max := calculateMax(frame, col.column, numRows)
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, []float64{max}))
		default:
			// Non-aggregated column - take first value
			field := findField(frame, col.column)
			if field != nil && numRows > 0 {
				newField := data.NewFieldFromFieldType(field.Type(), 1)
				newField.Name = alias
				newField.Set(0, field.At(0))
				newFrame.Fields = append(newFrame.Fields, newField)
			}
		}
	}

	return newFrame, nil
}

// aggregateByGroup aggregates with GROUP BY
func aggregateByGroup(frame *data.Frame, aggCols []aggregationColumn, groupByColumns []string, numRows int) (*data.Frame, error) {
	// Build groups - map of group key to row indices
	groups := make(map[string][]int)
	groupOrder := make([]string, 0)

	for i := 0; i < numRows; i++ {
		keyParts := make([]string, len(groupByColumns))
		for j, colName := range groupByColumns {
			field := findField(frame, colName)
			if field != nil {
				keyParts[j] = formatValue(field.At(i))
			}
		}
		key := strings.Join(keyParts, "|")
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], i)
	}

	newFrame := data.NewFrame(frame.Name)

	// Create fields for each column
	for _, col := range aggCols {
		alias := col.alias
		if alias == "" {
			if col.function != "" {
				alias = strings.ToLower(col.function)
			} else {
				alias = col.column
			}
		}

		switch col.function {
		case "COUNT":
			values := make([]int64, len(groupOrder))
			for i, key := range groupOrder {
				values[i] = int64(len(groups[key]))
			}
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, values))
		case "AVG":
			values := make([]float64, len(groupOrder))
			for i, key := range groupOrder {
				values[i] = calculateAvgForRows(frame, col.column, groups[key])
			}
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, values))
		case "SUM":
			values := make([]float64, len(groupOrder))
			for i, key := range groupOrder {
				values[i] = calculateSumForRows(frame, col.column, groups[key])
			}
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, values))
		case "MIN":
			values := make([]float64, len(groupOrder))
			for i, key := range groupOrder {
				values[i] = calculateMinForRows(frame, col.column, groups[key])
			}
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, values))
		case "MAX":
			values := make([]float64, len(groupOrder))
			for i, key := range groupOrder {
				values[i] = calculateMaxForRows(frame, col.column, groups[key])
			}
			newFrame.Fields = append(newFrame.Fields, data.NewField(alias, nil, values))
		default:
			// Regular column - take first value from each group
			field := findField(frame, col.column)
			if field != nil {
				newField := data.NewFieldFromFieldType(field.Type(), len(groupOrder))
				newField.Name = alias
				for i, key := range groupOrder {
					rows := groups[key]
					if len(rows) > 0 {
						newField.Set(i, field.At(rows[0]))
					}
				}
				newFrame.Fields = append(newFrame.Fields, newField)
			}
		}
	}

	return newFrame, nil
}

// Helper functions for aggregation calculations
func calculateAvg(frame *data.Frame, colName string, numRows int) float64 {
	rows := make([]int, numRows)
	for i := 0; i < numRows; i++ {
		rows[i] = i
	}
	return calculateAvgForRows(frame, colName, rows)
}

func calculateAvgForRows(frame *data.Frame, colName string, rows []int) float64 {
	field := findField(frame, colName)
	if field == nil || len(rows) == 0 {
		return 0
	}
	var sum float64
	count := 0
	for _, i := range rows {
		val := getNumericValue(field.At(i))
		if val != nil {
			sum += *val
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func calculateSum(frame *data.Frame, colName string, numRows int) float64 {
	rows := make([]int, numRows)
	for i := 0; i < numRows; i++ {
		rows[i] = i
	}
	return calculateSumForRows(frame, colName, rows)
}

func calculateSumForRows(frame *data.Frame, colName string, rows []int) float64 {
	field := findField(frame, colName)
	if field == nil {
		return 0
	}
	var sum float64
	for _, i := range rows {
		val := getNumericValue(field.At(i))
		if val != nil {
			sum += *val
		}
	}
	return sum
}

func calculateMin(frame *data.Frame, colName string, numRows int) float64 {
	rows := make([]int, numRows)
	for i := 0; i < numRows; i++ {
		rows[i] = i
	}
	return calculateMinForRows(frame, colName, rows)
}

func calculateMinForRows(frame *data.Frame, colName string, rows []int) float64 {
	field := findField(frame, colName)
	if field == nil || len(rows) == 0 {
		return 0
	}
	var min *float64
	for _, i := range rows {
		val := getNumericValue(field.At(i))
		if val != nil {
			if min == nil || *val < *min {
				min = val
			}
		}
	}
	if min == nil {
		return 0
	}
	return *min
}

func calculateMax(frame *data.Frame, colName string, numRows int) float64 {
	rows := make([]int, numRows)
	for i := 0; i < numRows; i++ {
		rows[i] = i
	}
	return calculateMaxForRows(frame, colName, rows)
}

func calculateMaxForRows(frame *data.Frame, colName string, rows []int) float64 {
	field := findField(frame, colName)
	if field == nil || len(rows) == 0 {
		return 0
	}
	var max *float64
	for _, i := range rows {
		val := getNumericValue(field.At(i))
		if val != nil {
			if max == nil || *val > *max {
				max = val
			}
		}
	}
	if max == nil {
		return 0
	}
	return *max
}

// formatValue dereferences pointer values for proper string representation
func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	switch val := v.(type) {
	case *string:
		if val == nil {
			return "<nil>"
		}
		return *val
	case *int64:
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d", *val)
	case *float64:
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%g", *val)
	case *uint64:
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d", *val)
	case *bool:
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%v", *val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getNumericValue(v interface{}) *float64 {
	switch val := v.(type) {
	case float64:
		return &val
	case *float64:
		return val
	case float32:
		f := float64(val)
		return &f
	case *float32:
		if val != nil {
			f := float64(*val)
			return &f
		}
	case int64:
		f := float64(val)
		return &f
	case *int64:
		if val != nil {
			f := float64(*val)
			return &f
		}
	case int32:
		f := float64(val)
		return &f
	case *int32:
		if val != nil {
			f := float64(*val)
			return &f
		}
	case int:
		f := float64(val)
		return &f
	case *int:
		if val != nil {
			f := float64(*val)
			return &f
		}
	}
	return nil
}

// parseColumnList parses a comma-separated list of column names
func parseColumnList(columnsStr string) []string {
	var columns []string
	parts := strings.Split(columnsStr, ",")
	for _, part := range parts {
		col := strings.TrimSpace(part)
		// Remove any aliases (AS ...)
		if idx := strings.Index(strings.ToUpper(col), " AS "); idx != -1 {
			col = strings.TrimSpace(col[:idx])
		}
		// Remove quotes if present
		col = strings.Trim(col, `"'`)
		col = strings.Trim(col, "`")
		if col != "" && col != "*" {
			columns = append(columns, col)
		}
	}
	return columns
}

// selectColumns creates a new frame with only the specified columns
func selectColumns(frame *data.Frame, columns []string) (*data.Frame, error) {
	newFrame := data.NewFrame(frame.Name)

	for _, colName := range columns {
		field := findField(frame, colName)
		if field != nil {
			newFrame.Fields = append(newFrame.Fields, field)
		}
	}

	if len(newFrame.Fields) == 0 {
		return nil, fmt.Errorf("no matching columns found: %v", columns)
	}

	return newFrame, nil
}

// findField finds a field by name (case-insensitive)
func findField(frame *data.Frame, name string) *data.Field {
	name = strings.ToLower(strings.TrimSpace(name))
	// Remove quotes
	name = strings.Trim(name, `"'`)

	for _, field := range frame.Fields {
		fieldName := strings.ToLower(field.Name)

		// Direct match
		if fieldName == name {
			return field
		}

		// Match with dots replaced by underscores (sepal.width == sepal_width)
		if strings.ReplaceAll(fieldName, ".", "_") == strings.ReplaceAll(name, ".", "_") {
			return field
		}

		// Match without any separators
		normalizedField := strings.ReplaceAll(strings.ReplaceAll(fieldName, ".", ""), "_", "")
		normalizedName := strings.ReplaceAll(strings.ReplaceAll(name, ".", ""), "_", "")
		if normalizedField == normalizedName {
			return field
		}
	}
	return nil
}

// applyWhereClause applies a simple WHERE clause
func applyWhereClause(frame *data.Frame, whereClause string) (*data.Frame, error) {
	numRows, _ := frame.RowLen()
	if numRows == 0 {
		return frame, nil
	}

	// Parse simple conditions (column op value)
	conditions := parseConditions(whereClause)
	if len(conditions) == 0 {
		return frame, nil
	}

	// Find matching rows
	matchingRows := make([]int, 0)
	for i := 0; i < numRows; i++ {
		if evaluateConditions(frame, i, conditions) {
			matchingRows = append(matchingRows, i)
		}
	}

	return filterRows(frame, matchingRows)
}

type condition struct {
	column   string
	operator string
	value    string
	logic    string // AND, OR
}

// parseConditions parses WHERE clause into conditions
func parseConditions(whereClause string) []condition {
	var conditions []condition

	// Split by AND/OR (simplified - doesn't handle parentheses)
	parts := regexp.MustCompile(`(?i)\s+(AND|OR)\s+`).Split(whereClause, -1)
	logics := regexp.MustCompile(`(?i)\s+(AND|OR)\s+`).FindAllString(whereClause, -1)

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse comparison: column op value
		ops := []string{">=", "<=", "!=", "<>", "=", ">", "<", " LIKE ", " like "}
		for _, op := range ops {
			idx := strings.Index(part, op)
			if idx != -1 {
				col := strings.TrimSpace(part[:idx])
				val := strings.TrimSpace(part[idx+len(op):])
				// Remove quotes from value
				val = strings.Trim(val, "'\"")

				logic := ""
				if i > 0 && i-1 < len(logics) {
					logic = strings.TrimSpace(strings.ToUpper(logics[i-1]))
				}

				conditions = append(conditions, condition{
					column:   col,
					operator: strings.TrimSpace(strings.ToUpper(op)),
					value:    val,
					logic:    logic,
				})
				break
			}
		}
	}

	return conditions
}

// evaluateConditions evaluates all conditions for a row
func evaluateConditions(frame *data.Frame, row int, conditions []condition) bool {
	if len(conditions) == 0 {
		return true
	}

	result := evaluateCondition(frame, row, conditions[0])

	for i := 1; i < len(conditions); i++ {
		condResult := evaluateCondition(frame, row, conditions[i])
		if conditions[i].logic == "OR" {
			result = result || condResult
		} else {
			result = result && condResult
		}
	}

	return result
}

// evaluateCondition evaluates a single condition for a row
func evaluateCondition(frame *data.Frame, row int, cond condition) bool {
	field := findField(frame, cond.column)
	if field == nil {
		return true // Column not found, include row
	}

	if row >= field.Len() {
		return false
	}

	fieldValue := field.At(row)
	if fieldValue == nil {
		return false
	}

	switch cond.operator {
	case "=":
		return compareEqual(fieldValue, cond.value)
	case "!=", "<>":
		return !compareEqual(fieldValue, cond.value)
	case ">":
		return compareGreater(fieldValue, cond.value)
	case "<":
		return compareLess(fieldValue, cond.value)
	case ">=":
		return compareGreater(fieldValue, cond.value) || compareEqual(fieldValue, cond.value)
	case "<=":
		return compareLess(fieldValue, cond.value) || compareEqual(fieldValue, cond.value)
	case "LIKE":
		return compareLike(fieldValue, cond.value)
	}

	return true
}

// compareEqual compares two values for equality
func compareEqual(fieldValue any, compareValue string) bool {
	switch v := fieldValue.(type) {
	case string:
		return v == compareValue
	case *string:
		if v == nil {
			return false
		}
		return *v == compareValue
	case int64:
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return v == cv
	case *int64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return *v == cv
	case float64:
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return v == cv
	case *float64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return *v == cv
	}
	return fmt.Sprintf("%v", fieldValue) == compareValue
}

// compareGreater compares if field value is greater than compare value
func compareGreater(fieldValue any, compareValue string) bool {
	switch v := fieldValue.(type) {
	case int64:
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return v > cv
	case *int64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return *v > cv
	case float64:
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return v > cv
	case *float64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return *v > cv
	case string:
		return v > compareValue
	case *string:
		if v == nil {
			return false
		}
		return *v > compareValue
	}
	return false
}

// compareLess compares if field value is less than compare value
func compareLess(fieldValue any, compareValue string) bool {
	switch v := fieldValue.(type) {
	case int64:
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return v < cv
	case *int64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseInt(compareValue, 10, 64)
		return *v < cv
	case float64:
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return v < cv
	case *float64:
		if v == nil {
			return false
		}
		cv, _ := strconv.ParseFloat(compareValue, 64)
		return *v < cv
	case string:
		return v < compareValue
	case *string:
		if v == nil {
			return false
		}
		return *v < compareValue
	}
	return false
}

// compareLike compares using SQL LIKE pattern
func compareLike(fieldValue any, pattern string) bool {
	var strValue string
	switch v := fieldValue.(type) {
	case string:
		strValue = v
	case *string:
		if v == nil {
			return false
		}
		strValue = *v
	default:
		strValue = fmt.Sprintf("%v", fieldValue)
	}

	// Convert SQL LIKE pattern to regex
	pattern = strings.ReplaceAll(pattern, "%", ".*")
	pattern = strings.ReplaceAll(pattern, "_", ".")
	pattern = "^" + pattern + "$"

	matched, _ := regexp.MatchString("(?i)"+pattern, strValue)
	return matched
}

// filterRows creates a new frame with only the specified row indices
func filterRows(frame *data.Frame, rows []int) (*data.Frame, error) {
	newFrame := data.NewFrame(frame.Name)

	for _, field := range frame.Fields {
		newField := data.NewFieldFromFieldType(field.Type(), len(rows))
		newField.Name = field.Name
		newField.Labels = field.Labels

		for newIdx, oldIdx := range rows {
			newField.Set(newIdx, field.At(oldIdx))
		}

		newFrame.Fields = append(newFrame.Fields, newField)
	}

	return newFrame, nil
}

// applyOrderByClause applies ORDER BY clause
func applyOrderByClause(frame *data.Frame, orderByClause string) (*data.Frame, error) {
	numRows, _ := frame.RowLen()
	if numRows == 0 {
		return frame, nil
	}

	// Parse column and direction
	parts := strings.Fields(orderByClause)
	if len(parts) == 0 {
		return frame, nil
	}

	colName := parts[0]
	ascending := true
	if len(parts) > 1 && strings.ToUpper(parts[1]) == "DESC" {
		ascending = false
	}

	field := findField(frame, colName)
	if field == nil {
		return frame, nil
	}

	// Create index array and sort
	indices := make([]int, numRows)
	for i := range indices {
		indices[i] = i
	}

	sort.Slice(indices, func(i, j int) bool {
		vi := field.At(indices[i])
		vj := field.At(indices[j])
		less := compareLess(vi, fmt.Sprintf("%v", vj))
		if ascending {
			return less
		}
		return !less
	})

	return filterRows(frame, indices)
}

// limitFrame limits the frame to the specified number of rows
func limitFrame(frame *data.Frame, limit int) *data.Frame {
	numRows, _ := frame.RowLen()
	if numRows <= limit {
		return frame
	}

	indices := make([]int, limit)
	for i := range indices {
		indices[i] = i
	}

	newFrame, _ := filterRows(frame, indices)
	return newFrame
}
