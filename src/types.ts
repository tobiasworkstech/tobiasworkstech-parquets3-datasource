import { AwsAuthDataSourceJsonData, AwsAuthDataSourceSecureJsonData } from '@grafana/aws-sdk';
import { DataQuery } from '@grafana/schema';

export interface MyQuery extends DataQuery {
  path?: string;
  sqlQuery?: string;
  format?: 'table' | 'time_series';
}

export const DEFAULT_QUERY: Partial<MyQuery> = {
  path: '',
  sqlQuery: 'SELECT * FROM parquet',
};

/**
 * These are options configured for each DataSource instance
 * Extends AwsAuthDataSourceJsonData to get standard AWS auth fields
 */
export interface MyDataSourceOptions extends AwsAuthDataSourceJsonData {
  bucket?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData extends AwsAuthDataSourceSecureJsonData {}
