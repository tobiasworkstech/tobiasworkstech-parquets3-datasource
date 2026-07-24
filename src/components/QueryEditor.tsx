import React, { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import {
  InlineField,
  Input,
  Select,
  MultiSelect,
  Button,
  Icon,
  InlineSwitch,
  TextArea,
  useStyles2,
} from '@grafana/ui';
import { QueryEditorProps, SelectableValue, GrafanaTheme2 } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery, ColumnSelection, WhereCondition, OrderByItem } from '../types';
import { css } from '@emotion/css';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

type EditorMode = 'builder' | 'code';

const getStyles = (theme: GrafanaTheme2) => ({
  headerRow: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(2),
    padding: `${theme.spacing(1)} 0`,
    borderBottom: `1px solid ${theme.colors.border.weak}`,
    marginBottom: theme.spacing(2),
    flexWrap: 'wrap',
  }),
  headerSection: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  }),
  modeButtons: css({
    display: 'flex',
    marginLeft: 'auto',
  }),
  modeButton: css({
    padding: `${theme.spacing(0.5)} ${theme.spacing(1.5)}`,
    border: `1px solid ${theme.colors.border.medium}`,
    background: 'transparent',
    cursor: 'pointer',
    fontSize: theme.typography.bodySmall.fontSize,
    '&:first-of-type': {
      borderRadius: `${theme.shape.borderRadius(1)} 0 0 ${theme.shape.borderRadius(1)}`,
    },
    '&:last-of-type': {
      borderRadius: `0 ${theme.shape.borderRadius(1)} ${theme.shape.borderRadius(1)} 0`,
      borderLeft: 'none',
    },
  }),
  modeButtonActive: css({
    background: theme.colors.primary.main,
    color: theme.colors.primary.contrastText,
    borderColor: theme.colors.primary.main,
  }),
  sectionRow: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: `${theme.spacing(1)} 0`,
  }),
  columnRow: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: `${theme.spacing(0.5)} 0`,
  }),
  label: css({
    fontSize: theme.typography.bodySmall.fontSize,
    color: theme.colors.text.secondary,
    minWidth: theme.spacing(10),
  }),
  addButton: css({
    marginTop: theme.spacing(1),
  }),
  codeEditor: css({
    marginTop: theme.spacing(2),
  }),
  toggleLabel: css({
    fontSize: theme.typography.bodySmall.fontSize,
    marginRight: theme.spacing(0.5),
  }),
  fileSelector: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    flexWrap: 'wrap',
  }),
  filterInput: css({
    fontSize: theme.typography.bodySmall.fontSize,
  }),
  sectionContainer: css({
    marginTop: theme.spacing(2),
  }),
  sectionTitle: css({
    marginBottom: theme.spacing(1),
    fontWeight: theme.typography.fontWeightMedium,
    fontSize: theme.typography.body.fontSize,
  }),
  sectionTitleMuted: css({
    marginBottom: theme.spacing(1),
    fontWeight: theme.typography.fontWeightMedium,
    fontSize: theme.typography.body.fontSize,
    color: theme.colors.text.secondary,
  }),
  fileCount: css({
    fontSize: theme.typography.bodySmall.fontSize,
    color: theme.colors.text.secondary,
  }),
  sqlPreview: css({
    background: theme.colors.background.secondary,
    padding: theme.spacing(1.5),
    borderRadius: theme.shape.borderRadius(1),
    fontFamily: theme.typography.fontFamilyMonospace,
    fontSize: theme.typography.bodySmall.fontSize,
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-all',
  }),
  fileSelectorWrapper: css({
    marginBottom: theme.spacing(1),
  }),
  textArea: css({
    fontFamily: theme.typography.fontFamilyMonospace,
  }),
});

const formatOptions: Array<SelectableValue<string>> = [
  { label: 'Table', value: 'table' },
  { label: 'Time series', value: 'time_series' },
];

const aggregationOptions: Array<SelectableValue<string>> = [
  { label: 'None', value: '' },
  { label: 'COUNT', value: 'COUNT' },
  { label: 'SUM', value: 'SUM' },
  { label: 'AVG', value: 'AVG' },
  { label: 'MIN', value: 'MIN' },
  { label: 'MAX', value: 'MAX' },
];

const operatorOptions: Array<SelectableValue<string>> = [
  { label: '=', value: '=' },
  { label: '!=', value: '!=' },
  { label: '>', value: '>' },
  { label: '<', value: '<' },
  { label: '>=', value: '>=' },
  { label: '<=', value: '<=' },
  { label: 'LIKE', value: 'LIKE' },
  { label: 'IN', value: 'IN' },
  { label: 'IS NULL', value: 'IS NULL' },
  { label: 'IS NOT NULL', value: 'IS NOT NULL' },
];

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const styles = useStyles2(getStyles);
  const [editorMode, setEditorMode] = useState<EditorMode>(query.editorMode || 'builder');
  const [allFiles, setAllFiles] = useState<string[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [fileFilter, setFileFilter] = useState('');
  const [columns, setColumns] = useState<Array<SelectableValue<string>>>([]);
  const [columnSelections, setColumnSelections] = useState<ColumnSelection[]>(
    query.columnSelections?.length ? query.columnSelections : [{ column: '', alias: '', aggregation: '' }]
  );
  const [whereConditions, setWhereConditions] = useState<WhereCondition[]>(query.whereConditions || []);
  const [orderByItems, setOrderByItems] = useState<OrderByItem[]>(query.orderByItems || []);
  const [groupByColumns, setGroupByColumns] = useState<string[]>(query.groupByColumns || []);
  const [limit, setLimit] = useState<string>(query.queryLimit || '');
  const [loading, setLoading] = useState(false);

  // Date-range file pattern mode: expands `pathPattern`'s "{date}" placeholder
  // server-side into one file per day covered by the dashboard time-picker.
  const [useDatePattern, setUseDatePattern] = useState<boolean>(!!query.pathPattern);
  const [pathPattern, setPathPattern] = useState<string>(query.pathPattern || '');
  const [dateFormat, setDateFormat] = useState<string>(query.dateFormat || '');

  // Toggle states
  const [filterEnabled, setFilterEnabled] = useState(query.filterEnabled ?? false);
  const [groupEnabled, setGroupEnabled] = useState(query.groupEnabled ?? false);
  const [orderEnabled, setOrderEnabled] = useState(query.orderEnabled ?? false);
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const [format, setFormat] = useState<'table' | 'time_series'>(query.format || 'table');

  // Refs to avoid stale closures in the SQL sync effect
  const isFirstRender = useRef(true);
  const queryRef = useRef(query);
  queryRef.current = query;
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // Load files from S3 on mount
  useEffect(() => {
    const loadFiles = async () => {
      setFilesLoading(true);
      try {
        const result = await datasource.listFiles('', '*.parquet');
        setAllFiles(result);
      } catch {
        setAllFiles([]);
      } finally {
        setFilesLoading(false);
      }
    };

    if (datasource) {
      loadFiles();
    }
  }, [datasource]);

  // Filter files based on regex pattern
  const filteredFileOptions = useMemo(() => {
    let filtered = allFiles;

    if (fileFilter) {
      try {
        const regex = new RegExp(fileFilter, 'i');
        filtered = allFiles.filter((file) => regex.test(file));
      } catch {
        // Invalid regex, fall back to simple includes
        filtered = allFiles.filter((file) =>
          file.toLowerCase().includes(fileFilter.toLowerCase())
        );
      }
    }

    return filtered.map((file) => ({
      label: file,
      value: file,
    }));
  }, [allFiles, fileFilter]);

  // Load columns when the selected file(s) change. Schemas are assumed to be
  // identical across all selected files, so the first one is representative.
  // In date-pattern mode, preview columns using today's date substituted into
  // the pattern (the actual query will expand to one file per day at run time).
  const schemaPath = query.pathPattern
    ? query.pathPattern.replace('{date}', new Date().toISOString().slice(0, 10))
    : query.paths?.[0] || query.path;
  useEffect(() => {
    const loadColumns = async () => {
      if (!schemaPath) {
        setColumns([]);
        return;
      }

      try {
        setLoading(true);
        const result = await datasource.getSchema(schemaPath);
        const columnOptions = result.map((col: { name: string; type: string }) => ({
          label: `${col.name} (${col.type})`,
          value: col.name,
          description: col.type,
        }));
        setColumns(columnOptions);
      } catch {
        setColumns([]);
      } finally {
        setLoading(false);
      }
    };

    loadColumns();
  }, [schemaPath, datasource]);

  // Build SQL from builder options
  const buildSQL = useCallback(() => {
    if (!query.path && !query.paths?.length && !query.pathPattern) {
      return '';
    }

    let sql = 'SELECT ';

    // Columns with aggregations
    const colParts = columnSelections
      .filter((c) => c.column)
      .map((c) => {
        let col = `"${c.column}"`;
        if (c.aggregation) {
          col = `${c.aggregation}(${col})`;
        }
        if (c.alias) {
          col += ` AS "${c.alias}"`;
        }
        return col;
      });

    if (colParts.length > 0) {
      sql += colParts.join(', ');
    } else {
      sql += '*';
    }

    sql += ' FROM parquet';

    // WHERE
    if (filterEnabled && whereConditions.length > 0) {
      const conditions = whereConditions
        .filter((c) => c.column && (c.operator === 'IS NULL' || c.operator === 'IS NOT NULL' || c.value))
        .map((c) => {
          if (c.operator === 'IS NULL' || c.operator === 'IS NOT NULL') {
            return `"${c.column}" ${c.operator}`;
          }
          const value = c.operator === 'LIKE' || c.operator === 'IN' || isNaN(Number(c.value))
            ? (c.operator === 'IN' ? c.value : `'${c.value}'`)
            : c.value;
          return `"${c.column}" ${c.operator} ${value}`;
        });
      if (conditions.length > 0) {
        sql += ' WHERE ' + conditions.join(' AND ');
      }
    }

    // GROUP BY
    if (groupEnabled && groupByColumns.length > 0) {
      sql += ' GROUP BY ' + groupByColumns.map((c) => `"${c}"`).join(', ');
    }

    // ORDER BY
    if (orderEnabled && orderByItems.length > 0) {
      const orderParts = orderByItems
        .filter((o) => o.column)
        .map((o) => `"${o.column}" ${o.direction}`);
      if (orderParts.length > 0) {
        sql += ' ORDER BY ' + orderParts.join(', ');
      }
    }

    // LIMIT
    if (limit) {
      sql += ` LIMIT ${limit}`;
    }

    return sql;
  }, [columnSelections, whereConditions, orderByItems, groupByColumns, limit, query.path, query.paths, query.pathPattern, filterEnabled, groupEnabled, orderEnabled]);

  // Sync SQL and full builder state to query when builder options change.
  // Skips the first render to avoid overwriting a persisted query with default local state.
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }
    if (editorMode !== 'builder') {
      return;
    }
    const sql = buildSQL();
    onChangeRef.current({
      ...queryRef.current,
      sqlQuery: sql,
      format,
      editorMode,
      columnSelections,
      whereConditions,
      orderByItems,
      groupByColumns,
      queryLimit: limit,
      filterEnabled,
      groupEnabled,
      orderEnabled,
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editorMode, buildSQL, format, columnSelections, whereConditions, orderByItems, groupByColumns, limit, filterEnabled, groupEnabled, orderEnabled]);

  // Handles selection changes from the multi-file picker. `paths` holds every
  // selected file; `path` is kept in sync (comma-joined) for backward
  // compatibility with existing dashboards/template variables that read a
  // single path.
  const onPathsChange = (values: Array<SelectableValue<string>> | null) => {
    const paths = (values || []).map((v) => v.value!).filter(Boolean);
    onChange({ ...query, paths, path: paths.join(',') });
    // Reset builder state when the file selection changes
    setColumnSelections([{ column: '', alias: '', aggregation: '' }]);
    setWhereConditions([]);
    setOrderByItems([]);
    setGroupByColumns([]);
    setLimit('');
  };

  const onSqlQueryChange = (value: string) => {
    onChange({ ...query, sqlQuery: value });
  };

  // Toggles between explicit file selection and date-range pattern mode.
  // Switching modes clears the fields owned by the other mode so a stale
  // `pathPattern` doesn't silently override a freshly picked `path`/`paths`
  // (the backend prioritizes `pathPattern` when present).
  const onDatePatternToggle = () => {
    const next = !useDatePattern;
    setUseDatePattern(next);
    if (next) {
      onChange({ ...query, pathPattern, path: '', paths: [] });
    } else {
      onChange({ ...query, pathPattern: undefined, dateFormat: undefined });
    }
  };

  const onPathPatternChange = (value: string) => {
    setPathPattern(value);
    onChange({ ...query, pathPattern: value });
  };

  const onDateFormatChange = (value: string) => {
    setDateFormat(value);
    onChange({ ...query, dateFormat: value || undefined });
  };

  // Refresh files list
  const refreshFiles = async () => {
    setFilesLoading(true);
    try {
      const result = await datasource.listFiles('', '*.parquet');
      setAllFiles(result);
    } catch {
      // Failed to refresh files
    } finally {
      setFilesLoading(false);
    }
  };

  // Column selection handlers
  const addColumnSelection = () => {
    setColumnSelections([...columnSelections, { column: '', alias: '', aggregation: '' }]);
  };

  const updateColumnSelection = (index: number, field: keyof ColumnSelection, value: string) => {
    const updated = [...columnSelections];
    updated[index] = { ...updated[index], [field]: value };
    setColumnSelections(updated);
  };

  const removeColumnSelection = (index: number) => {
    if (columnSelections.length > 1) {
      setColumnSelections(columnSelections.filter((_, i) => i !== index));
    }
  };

  // WHERE condition handlers
  const addWhereCondition = () => {
    setWhereConditions([...whereConditions, { column: '', operator: '=', value: '' }]);
  };

  const updateWhereCondition = (index: number, field: keyof WhereCondition, value: string) => {
    const updated = [...whereConditions];
    updated[index] = { ...updated[index], [field]: value };
    setWhereConditions(updated);
  };

  const removeWhereCondition = (index: number) => {
    setWhereConditions(whereConditions.filter((_, i) => i !== index));
  };

  // ORDER BY handlers
  const addOrderByItem = () => {
    setOrderByItems([...orderByItems, { column: '', direction: 'ASC' }]);
  };

  const updateOrderByItem = (index: number, field: keyof OrderByItem, value: string) => {
    const updated = [...orderByItems];
    updated[index] = { ...updated[index], [field]: value as 'ASC' | 'DESC' };
    setOrderByItems(updated);
  };

  const removeOrderByItem = (index: number) => {
    setOrderByItems(orderByItems.filter((_, i) => i !== index));
  };

  const { sqlQuery } = query;
  const selectedPaths = query.paths?.length ? query.paths : query.path ? [query.path] : [];

  // File selector component (reused in both modes). Supports selecting
  // multiple parquet files to search across at once.
  const FileSelector = () => (
    <div className={styles.fileSelector}>
      <span className={styles.label}>Table</span>
      {useDatePattern ? (
        <>
          <Input
            value={pathPattern}
            onChange={(e) => onPathPatternChange(e.currentTarget.value)}
            placeholder="e.g. metrics-curated/dt={date}/data.parquet"
            width={45}
          />
          <Input
            value={dateFormat}
            onChange={(e) => onDateFormatChange(e.currentTarget.value)}
            placeholder="Date format (default 2006-01-02)"
            width={28}
          />
        </>
      ) : (
        <>
          <MultiSelect
            options={filteredFileOptions}
            value={selectedPaths.map((p) => ({ label: p, value: p }))}
            onChange={onPathsChange}
            placeholder="Select one or more parquet files..."
            isLoading={filesLoading}
            isClearable
            allowCustomValue
            onCreateOption={(v) => onPathsChange([...selectedPaths.map((p) => ({ label: p, value: p })), { label: v, value: v }])}
            width={40}
            noOptionsMessage="No files found"
          />
          <Input
            value={fileFilter}
            onChange={(e) => setFileFilter(e.currentTarget.value)}
            placeholder="Filter (regex)"
            width={15}
            className={styles.filterInput}
          />
          <Button
            variant="secondary"
            size="sm"
            onClick={refreshFiles}
            disabled={filesLoading}
            title="Refresh file list"
          >
            <Icon name="sync" />
          </Button>
          {allFiles.length > 0 && (
            <span className={styles.fileCount}>
              {filteredFileOptions.length} of {allFiles.length} files
            </span>
          )}
        </>
      )}
      <span className={styles.toggleLabel}>Match by time range</span>
      <InlineSwitch value={useDatePattern} onChange={onDatePatternToggle} transparent title="Auto-select daily files using the dashboard time range" />
    </div>
  );

  return (
    <div>
      {/* Header Row - PostgreSQL style */}
      <div className={styles.headerRow}>
        <div className={styles.headerSection}>
          <span className={styles.toggleLabel}>Format</span>
          <Select
            options={formatOptions}
            value={formatOptions.find((f) => f.value === format)}
            onChange={(v) => {
              const newFormat = (v?.value as 'table' | 'time_series') || 'table';
              setFormat(newFormat);
              onChange({ ...query, format: newFormat, editorMode });
            }}
            width={15}
          />
        </div>

        <div className={styles.headerSection}>
          <span className={styles.toggleLabel}>Filter</span>
          <InlineSwitch
            value={filterEnabled}
            onChange={() => setFilterEnabled(!filterEnabled)}
            transparent
          />
        </div>

        <div className={styles.headerSection}>
          <span className={styles.toggleLabel}>Group</span>
          <InlineSwitch
            value={groupEnabled}
            onChange={() => setGroupEnabled(!groupEnabled)}
            transparent
          />
        </div>

        <div className={styles.headerSection}>
          <span className={styles.toggleLabel}>Order</span>
          <InlineSwitch
            value={orderEnabled}
            onChange={() => setOrderEnabled(!orderEnabled)}
            transparent
          />
        </div>

        <div className={styles.headerSection}>
          <span className={styles.toggleLabel}>Preview</span>
          <InlineSwitch
            value={previewEnabled}
            onChange={() => setPreviewEnabled(!previewEnabled)}
            transparent
          />
        </div>

        <Button variant="primary" size="sm" onClick={onRunQuery}>
          Run query
        </Button>

        <div className={styles.modeButtons}>
          <button
            className={`${styles.modeButton} ${editorMode === 'builder' ? styles.modeButtonActive : ''}`}
            onClick={() => {
              setEditorMode('builder');
              onChange({ ...query, editorMode: 'builder' });
            }}
          >
            Builder
          </button>
          <button
            className={`${styles.modeButton} ${editorMode === 'code' ? styles.modeButtonActive : ''}`}
            onClick={() => {
              setEditorMode('code');
              onChange({ ...query, editorMode: 'code' });
            }}
          >
            Code
          </button>
        </div>
      </div>

      {editorMode === 'builder' ? (
        <>
          {/* Table (File) Selector */}
          <div className={styles.sectionRow}>
            <FileSelector />
          </div>

          {/* Column Selections */}
          <div className={styles.sectionContainer}>
            <div className={styles.sectionTitle}>Columns</div>
            {columnSelections.map((selection, index) => (
              <div key={index} className={styles.columnRow}>
                <Select
                  options={aggregationOptions}
                  value={aggregationOptions.find((a) => a.value === selection.aggregation) || aggregationOptions[0]}
                  onChange={(v) => updateColumnSelection(index, 'aggregation', v?.value || '')}
                  placeholder="Aggregation"
                  width={15}
                />
                <Select
                  options={columns}
                  value={selection.column ? columns.find((c) => c.value === selection.column) || { label: selection.column, value: selection.column } : null}
                  onChange={(v) => updateColumnSelection(index, 'column', v?.value || '')}
                  isLoading={loading}
                  placeholder="Column"
                  width={25}
                  isClearable
                />
                <Input
                  value={selection.alias}
                  onChange={(e) => updateColumnSelection(index, 'alias', e.currentTarget.value)}
                  placeholder="Alias (optional)"
                  width={20}
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => removeColumnSelection(index)}
                  disabled={columnSelections.length === 1}
                >
                  <Icon name="trash-alt" />
                </Button>
              </div>
            ))}
            <Button variant="secondary" size="sm" onClick={addColumnSelection} className={styles.addButton}>
              <Icon name="plus" /> Add column
            </Button>
          </div>

          {/* Filter (WHERE) Section */}
          {filterEnabled && (
            <div className={styles.sectionContainer}>
              <div className={styles.sectionTitle}>Filter (WHERE)</div>
              {whereConditions.map((condition, index) => (
                <div key={index} className={styles.columnRow}>
                  <Select
                    options={columns}
                    value={condition.column ? columns.find((c) => c.value === condition.column) || { label: condition.column, value: condition.column } : null}
                    onChange={(v) => updateWhereCondition(index, 'column', v?.value || '')}
                    placeholder="Column"
                    width={20}
                  />
                  <Select
                    options={operatorOptions}
                    value={operatorOptions.find((o) => o.value === condition.operator)}
                    onChange={(v) => updateWhereCondition(index, 'operator', v?.value || '=')}
                    width={15}
                  />
                  {condition.operator !== 'IS NULL' && condition.operator !== 'IS NOT NULL' && (
                    <Input
                      value={condition.value}
                      onChange={(e) => updateWhereCondition(index, 'value', e.currentTarget.value)}
                      placeholder="Value"
                      width={20}
                    />
                  )}
                  <Button variant="secondary" size="sm" onClick={() => removeWhereCondition(index)}>
                    <Icon name="trash-alt" />
                  </Button>
                </div>
              ))}
              <Button variant="secondary" size="sm" onClick={addWhereCondition} className={styles.addButton}>
                <Icon name="plus" /> Add filter
              </Button>
            </div>
          )}

          {/* Group By Section */}
          {groupEnabled && (
            <div className={styles.sectionContainer}>
              <div className={styles.sectionTitle}>Group By</div>
              <div className={styles.sectionRow}>
                <Select
                  options={columns}
                  value={groupByColumns.map((c) => ({ label: c, value: c }))}
                  onChange={(values) => setGroupByColumns(values ? values.map((v: SelectableValue<string>) => v.value!) : [])}
                  isMulti
                  isClearable
                  isLoading={loading}
                  placeholder="Select columns to group by..."
                  width={40}
                />
              </div>
            </div>
          )}

          {/* Order By Section */}
          {orderEnabled && (
            <div className={styles.sectionContainer}>
              <div className={styles.sectionTitle}>Order By</div>
              {orderByItems.map((item, index) => (
                <div key={index} className={styles.columnRow}>
                  <Select
                    options={columns}
                    value={item.column ? columns.find((c) => c.value === item.column) || { label: item.column, value: item.column } : null}
                    onChange={(v) => updateOrderByItem(index, 'column', v?.value || '')}
                    placeholder="Column"
                    width={25}
                  />
                  <Select
                    options={[
                      { label: 'ASC', value: 'ASC' },
                      { label: 'DESC', value: 'DESC' },
                    ]}
                    value={{ label: item.direction, value: item.direction }}
                    onChange={(v) => updateOrderByItem(index, 'direction', v?.value || 'ASC')}
                    width={12}
                  />
                  <Button variant="secondary" size="sm" onClick={() => removeOrderByItem(index)}>
                    <Icon name="trash-alt" />
                  </Button>
                </div>
              ))}
              <Button variant="secondary" size="sm" onClick={addOrderByItem} className={styles.addButton}>
                <Icon name="plus" /> Add order
              </Button>
            </div>
          )}

          {/* Limit */}
          <div className={styles.sectionContainer}>
            <InlineField label="Limit" labelWidth={8}>
              <Input
                value={limit}
                onChange={(e) => setLimit(e.currentTarget.value)}
                placeholder="e.g., 100"
                type="number"
                width={15}
              />
            </InlineField>
          </div>

          {/* Generated SQL Preview */}
          {previewEnabled && (
            <div className={styles.sectionContainer}>
              <div className={styles.sectionTitleMuted}>Generated SQL</div>
              <div className={styles.sqlPreview}>
                {buildSQL() || 'Select a table to generate SQL'}
              </div>
            </div>
          )}
        </>
      ) : (
        /* Code Mode */
        <div className={styles.codeEditor}>
          <div className={styles.fileSelectorWrapper}>
            <FileSelector />
          </div>
          <div className={styles.sectionContainer}>
            <div className={styles.sectionTitle}>SQL Query</div>
            <TextArea
              value={sqlQuery || ''}
              onChange={(e) => onSqlQueryChange(e.currentTarget.value)}
              onBlur={onRunQuery}
              placeholder="SELECT * FROM parquet WHERE column > 10 ORDER BY column DESC LIMIT 100"
              rows={8}
              className={styles.textArea}
            />
          </div>
        </div>
      )}
    </div>
  );
}
