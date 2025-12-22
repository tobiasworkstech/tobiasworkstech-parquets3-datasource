import React, { ChangeEvent } from 'react';
import { InlineField, Input, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onPathChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, path: event.target.value });
  };

  const { path } = query;

  return (
    <Stack gap={0}>
      <InlineField label="Parquet File Path" labelWidth={20} tooltip="Path to the .parquet file within the bucket">
        <Input
          onChange={onPathChange}
          value={path || ''}
          placeholder="path/to/data.parquet"
          width={60}
          onBlur={onRunQuery}
        />
      </InlineField>
    </Stack>
  );
}
