import React, { ChangeEvent } from 'react';
import { ConnectionConfig } from '@grafana/aws-sdk';
import { InlineField, Input } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData } = options;

  const onBucketChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        bucket: event.target.value,
      },
    });
  };

  return (
    <>
      <ConnectionConfig {...props} />
      <InlineField label="Default Bucket" labelWidth={28} tooltip="Default S3 bucket name for parquet files">
        <Input
          onChange={onBucketChange}
          value={jsonData.bucket || ''}
          placeholder="my-parquet-bucket"
          width={40}
        />
      </InlineField>
    </>
  );
}
