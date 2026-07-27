import { AwsAuthDataSourceJsonData, AwsAuthDataSourceSecureJsonData } from '@grafana/aws-sdk';
import { DataQuery } from '@grafana/schema';

export interface ColumnSelection {
  column: string;
  alias: string;
  aggregation: string;
}

export interface WhereCondition {
  column: string;
  operator: string;
  value: string;
}

export interface OrderByItem {
  column: string;
  direction: 'ASC' | 'DESC';
}

export interface MyQuery extends DataQuery {
  path?: string;
  paths?: string[];
  /**
   * A path template containing a "{date}" placeholder (e.g.
   * "metrics-curated/dt={date}/data.parquet") that is expanded, server-side,
   * into one file per day covered by the dashboard's time range. Lets a
   * top-of-dashboard time-picker drive which daily-partitioned Parquet files
   * are queried, without manually selecting files. Takes priority over
   * `path`/`paths` when set.
   */
  pathPattern?: string;
  /** Go reference-time layout used to format "{date}" in `pathPattern`. Defaults to "2006-01-02" (YYYY-MM-DD). */
  dateFormat?: string;
  sqlQuery?: string;
  format?: 'table' | 'time_series';
  editorMode?: 'builder' | 'code';
  columnSelections?: ColumnSelection[];
  whereConditions?: WhereCondition[];
  orderByItems?: OrderByItem[];
  groupByColumns?: string[];
  queryLimit?: string;
  filterEnabled?: boolean;
  groupEnabled?: boolean;
  orderEnabled?: boolean;
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
