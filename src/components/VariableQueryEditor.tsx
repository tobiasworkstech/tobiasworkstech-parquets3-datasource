import React, { ChangeEvent } from 'react';
import { InlineField, Input, Select, Stack, TextArea } from '@grafana/ui';
import { SelectableValue } from '@grafana/data';

export interface VariableQuery {
  queryType?: 'variable' | 'prefixes' | 'files' | 'sql';
  variableQueryType?: 'prefixes' | 'files' | 'sql';
  prefix?: string;
  filePattern?: string;
  path?: string;
  sqlQuery?: string;
}

interface Props {
  query: VariableQuery;
  onChange: (query: VariableQuery, definition: string) => void;
}

const variableQueryTypeOptions: Array<SelectableValue<string>> = [
  { label: 'List Files', value: 'files', description: 'List parquet files matching a pattern' },
  { label: 'List Prefixes/Folders', value: 'prefixes', description: 'List prefixes (folders) in the bucket' },
  { label: 'SQL Query', value: 'sql', description: 'Execute SQL query and use first column as values' },
];

export function VariableQueryEditor({ query, onChange }: Props) {
  const onVariableQueryTypeChange = (value: SelectableValue<string>) => {
    const newQuery: VariableQuery = {
      ...query,
      queryType: 'variable',
      variableQueryType: (value.value as VariableQuery['variableQueryType']) || 'files',
    };
    onChange(newQuery, formatDefinition(newQuery));
  };

  const onPrefixChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newQuery: VariableQuery = {
      ...query,
      queryType: 'variable',
      prefix: event.target.value,
    };
    onChange(newQuery, formatDefinition(newQuery));
  };

  const onFilePatternChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newQuery: VariableQuery = {
      ...query,
      queryType: 'variable',
      filePattern: event.target.value,
    };
    onChange(newQuery, formatDefinition(newQuery));
  };

  const onPathChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newQuery: VariableQuery = {
      ...query,
      queryType: 'variable',
      path: event.target.value,
    };
    onChange(newQuery, formatDefinition(newQuery));
  };

  const onSqlQueryChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const newQuery: VariableQuery = {
      ...query,
      queryType: 'variable',
      sqlQuery: event.target.value,
    };
    onChange(newQuery, formatDefinition(newQuery));
  };

  const formatDefinition = (q: VariableQuery): string => {
    switch (q.variableQueryType) {
      case 'prefixes':
        return `prefixes(${q.prefix || ''})`;
      case 'files':
        return `files(${q.prefix || ''}, ${q.filePattern || '*.parquet'})`;
      case 'sql':
        return `sql(${q.path || ''}, ${q.sqlQuery || ''})`;
      default:
        return `files(${q.prefix || ''}, ${q.filePattern || '*.parquet'})`;
    }
  };

  return (
    <Stack direction="column" gap={1}>
      <InlineField label="Query Type" labelWidth={20}>
        <Select
          options={variableQueryTypeOptions}
          value={variableQueryTypeOptions.find((o) => o.value === query.variableQueryType) || variableQueryTypeOptions[0]}
          onChange={onVariableQueryTypeChange}
          width={40}
        />
      </InlineField>

      {(query.variableQueryType === 'files' || query.variableQueryType === 'prefixes' || !query.variableQueryType) && (
        <InlineField label="Prefix" labelWidth={20} tooltip="S3 prefix (folder path) to search within">
          <Input
            onChange={onPrefixChange}
            value={query.prefix || ''}
            placeholder="data/2024/"
            width={40}
          />
        </InlineField>
      )}

      {(query.variableQueryType === 'files' || !query.variableQueryType) && (
        <InlineField label="File Pattern" labelWidth={20} tooltip="Glob pattern to match files (e.g., *.parquet)">
          <Input
            onChange={onFilePatternChange}
            value={query.filePattern || ''}
            placeholder="*.parquet"
            width={40}
          />
        </InlineField>
      )}

      {query.variableQueryType === 'sql' && (
        <>
          <InlineField label="Parquet File" labelWidth={20} tooltip="Path to the parquet file in S3">
            <Input
              onChange={onPathChange}
              value={query.path || ''}
              placeholder="data.parquet"
              width={40}
            />
          </InlineField>
          <InlineField label="SQL Query" labelWidth={20} tooltip="SQL query - first column will be used as variable values">
            <TextArea
              onChange={onSqlQueryChange}
              value={query.sqlQuery || ''}
              placeholder="SELECT DISTINCT category FROM parquet"
              rows={3}
              cols={50}
            />
          </InlineField>
        </>
      )}
    </Stack>
  );
}
