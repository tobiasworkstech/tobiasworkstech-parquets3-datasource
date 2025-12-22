import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface MyQuery extends DataQuery {
  path?: string;
}

export const DEFAULT_QUERY: Partial<MyQuery> = {
  path: '',
};

/**
 * These are options configured for each DataSource instance
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  region?: string;
  bucket?: string;
  endpoint?: string;
  useSSL?: boolean;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  accessKey?: string;
  secretKey?: string;
}
