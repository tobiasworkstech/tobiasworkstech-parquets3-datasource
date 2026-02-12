import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  InlineField,
  Input,
  Select,
  Button,
  Icon,
  InlineSwitch,
  TextArea,
} from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';
import { css } from '@emotion/css';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

type EditorMode = 'builder' | 'code';

interface ColumnSelection {
  column: string;
  alias: string;
  aggregation: string;
}

interface WhereCondition {
  column: string;
  operator: string;
  value: string;
}

interface OrderByItem {
  column: string;
  direction: 'ASC' | 'DESC';
}

const styles = {
  headerRow: css`
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-weak);
    margin-bottom: 16px;
    flex-wrap: wrap;
  `,
  headerSection: css`
    display: flex;
    align-items: center;
    gap: 8px;
  `,
  modeButtons: css`
    display: flex;
    margin-left: auto;
  `,
  modeButton: css`
    padding: 4px 12px;
    border: 1px solid var(--border-medium);
    background: transparent;
    cursor: pointer;
    font-size: 12px;
    &:first-of-type {
      border-radius: 4px 0 0 4px;
    }
    &:last-of-type {
      border-radius: 0 4px 4px 0;
      border-left: none;
    }
  `,
  modeButtonActive: css`
    background: var(--primary-main);
    color: white;
    border-color: var(--primary-main);
  `,
  sectionRow: css`
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 0;
  `,
  columnRow: css`
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
  `,
  label: css`
    font-size: 12px;
    color: var(--text-secondary);
    min-width: 80px;
  `,
  addButton: css`
    margin-top: 8px;
  `,
  codeEditor: css`
    margin-top: 16px;
  `,
  toggleLabel: css`
    font-size: 12px;
    margin-right: 4px;
  `,
  fileSelector: css`
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  `,
  filterInput: css`
    font-size: 12px;
  `,
};

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
  const [editorMode, setEditorMode] = useState<EditorMode>('builder');
  const [allFiles, setAllFiles] = useState<string[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [fileFilter, setFileFilter] = useState('');
  const [columns, setColumns] = useState<Array<SelectableValue<string>>>([]);
  const [columnSelections, setColumnSelections] = useState<ColumnSelection[]>([{ column: '', alias: '', aggregation: '' }]);
  const [whereConditions, setWhereConditions] = useState<WhereCondition[]>([]);
  const [orderByItems, setOrderByItems] = useState<OrderByItem[]>([]);
  const [groupByColumns, setGroupByColumns] = useState<string[]>([]);
  const [limit, setLimit] = useState<string>('');
  const [loading, setLoading] = useState(false);

  // Toggle states
  const [filterEnabled, setFilterEnabled] = useState(false);
  const [groupEnabled, setGroupEnabled] = useState(false);
  const [orderEnabled, setOrderEnabled] = useState(false);
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const [format, setFormat] = useState<'table' | 'time_series'>('table');

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

  // Load columns when file changes
  useEffect(() => {
    const loadColumns = async () => {
      if (!query.path) {
        setColumns([]);
        return;
      }

      try {
        setLoading(true);
        const result = await datasource.getSchema(query.path);
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
  }, [query.path, datasource]);

  // Build SQL from builder options
  const buildSQL = useCallback(() => {
    if (!query.path) {
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
  }, [columnSelections, whereConditions, orderByItems, groupByColumns, limit, query.path, filterEnabled, groupEnabled, orderEnabled]);

  // Sync SQL when builder options change
  useEffect(() => {
    if (editorMode === 'builder') {
      const sql = buildSQL();
      if (sql !== query.sqlQuery) {
        onChange({ ...query, sqlQuery: sql, format });
      }
    }
  }, [editorMode, buildSQL, query, onChange, format]);

  const onPathChange = (value: SelectableValue<string>) => {
    onChange({ ...query, path: value?.value || '' });
    // Reset builder state when file changes
    setColumnSelections([{ column: '', alias: '', aggregation: '' }]);
    setWhereConditions([]);
    setOrderByItems([]);
    setGroupByColumns([]);
    setLimit('');
  };

  const onSqlQueryChange = (value: string) => {
    onChange({ ...query, sqlQuery: value });
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

  const { path, sqlQuery } = query;

  // File selector component (reused in both modes)
  const FileSelector = () => (
    <div className={styles.fileSelector}>
      <span className={styles.label}>Table</span>
      <Select
        options={filteredFileOptions}
        value={path ? { label: path, value: path } : null}
        onChange={onPathChange}
        placeholder="Select parquet file..."
        isLoading={filesLoading}
        isClearable
        allowCustomValue
        onCreateOption={(v) => onChange({ ...query, path: v })}
        width={30}
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
        <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
          {filteredFileOptions.length} of {allFiles.length} files
        </span>
      )}
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
            onChange={(v) => setFormat((v?.value as 'table' | 'time_series') || 'table')}
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
            onClick={() => setEditorMode('builder')}
          >
            Builder
          </button>
          <button
            className={`${styles.modeButton} ${editorMode === 'code' ? styles.modeButtonActive : ''}`}
            onClick={() => setEditorMode('code')}
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
          <div style={{ marginTop: '16px' }}>
            <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px' }}>Columns</div>
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
            <div style={{ marginTop: '16px' }}>
              <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px' }}>Filter (WHERE)</div>
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
            <div style={{ marginTop: '16px' }}>
              <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px' }}>Group By</div>
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
            <div style={{ marginTop: '16px' }}>
              <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px' }}>Order By</div>
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
          <div style={{ marginTop: '16px' }}>
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
            <div style={{ marginTop: '16px' }}>
              <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px', color: 'var(--text-secondary)' }}>
                Generated SQL
              </div>
              <div style={{
                background: 'var(--background-secondary)',
                padding: '12px',
                borderRadius: '4px',
                fontFamily: 'monospace',
                fontSize: '12px',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all'
              }}>
                {buildSQL() || 'Select a table to generate SQL'}
              </div>
            </div>
          )}
        </>
      ) : (
        /* Code Mode */
        <div className={styles.codeEditor}>
          <div style={{ marginBottom: '8px' }}>
            <FileSelector />
          </div>
          <div style={{ marginTop: '16px' }}>
            <div style={{ marginBottom: '8px', fontWeight: 500, fontSize: '14px' }}>SQL Query</div>
            <TextArea
              value={sqlQuery || ''}
              onChange={(e) => onSqlQueryChange(e.currentTarget.value)}
              onBlur={onRunQuery}
              placeholder="SELECT * FROM parquet WHERE column > 10 ORDER BY column DESC LIMIT 100"
              rows={8}
              style={{ fontFamily: 'monospace' }}
            />
          </div>
        </div>
      )}
    </div>
  );
}
