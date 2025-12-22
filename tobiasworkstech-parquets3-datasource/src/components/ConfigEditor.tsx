import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> { }

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onRegionChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        region: event.target.value,
      },
    });
  };

  const onBucketChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        bucket: event.target.value,
      },
    });
  };

  const onAccessKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        accessKey: event.target.value,
      },
    });
  };

  const onSecretKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        secretKey: event.target.value,
      },
    });
  };

  const onResetAccessKey = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        accessKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        accessKey: '',
      },
    });
  };

  const onResetSecretKey = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        secretKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        secretKey: '',
      },
    });
  };

  const onEndpointChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        endpoint: event.target.value,
      },
    });
  };

  return (
    <>
      <InlineField label="Region" labelWidth={14}>
        <Input
          onChange={onRegionChange}
          value={jsonData.region || ''}
          placeholder="us-east-1"
          width={40}
        />
      </InlineField>
      <InlineField label="Bucket" labelWidth={14}>
        <Input
          onChange={onBucketChange}
          value={jsonData.bucket || ''}
          placeholder="my-parquet-bucket"
          width={40}
        />
      </InlineField>
      <InlineField label="Endpoint" labelWidth={14} tooltip="Custom S3 endpoint (e.g. for Minio)">
        <Input
          onChange={onEndpointChange}
          value={jsonData.endpoint || ''}
          placeholder="http://minio:9000"
          width={40}
        />
      </InlineField>
      <InlineField label="Access Key" labelWidth={14}>
        <SecretInput
          isConfigured={secureJsonFields.accessKey}
          value={secureJsonData?.accessKey || ''}
          placeholder="AWS Access Key"
          width={40}
          onReset={onResetAccessKey}
          onChange={onAccessKeyChange}
        />
      </InlineField>
      <InlineField label="Secret Key" labelWidth={14}>
        <SecretInput
          isConfigured={secureJsonFields.secretKey}
          value={secureJsonData?.secretKey || ''}
          placeholder="AWS Secret Key"
          width={40}
          onReset={onResetSecretKey}
          onChange={onSecretKeyChange}
        />
      </InlineField>
    </>
  );
}
