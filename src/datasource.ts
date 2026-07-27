import { DataSourceInstanceSettings, CoreApp, ScopedVars, MetricFindValue } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv, getBackendSrv } from '@grafana/runtime';

import { MyQuery, MyDataSourceOptions, DEFAULT_QUERY } from './types';
import { VariableQuery } from './components/VariableQueryEditor';

export class DataSource extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<MyQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: MyQuery, scopedVars: ScopedVars) {
    return {
      ...query,
      path: getTemplateSrv().replace(query.path, scopedVars),
      paths: query.paths?.map((p) => getTemplateSrv().replace(p, scopedVars)),
      pathPattern: query.pathPattern ? getTemplateSrv().replace(query.pathPattern, scopedVars) : query.pathPattern,
      // Interpolates dashboard variables as well as Grafana's built-in time-range
      // globals ($__from, $__to, ${__from:date:...}, etc.), so a WHERE clause like
      // "event_time >= $__from AND event_time <= $__to" is expanded to concrete
      // values before being sent to the backend's SQL executor.
      sqlQuery: query.sqlQuery ? getTemplateSrv().replace(query.sqlQuery, scopedVars) : query.sqlQuery,
    };
  }

  filterQuery(query: MyQuery): boolean {
    // if no path (or paths, or a date-range path pattern) has been provided, prevent the query from being executed
    return !!query.path || !!query.paths?.length || !!query.pathPattern;
  }

  /**
   * Supports template variable queries for listing S3 objects
   */
  async metricFindQuery(query: VariableQuery | string, options?: { scopedVars?: ScopedVars }): Promise<MetricFindValue[]> {
    // Handle legacy string queries
    if (typeof query === 'string') {
      return this.parseStringQuery(query);
    }

    // Handle structured variable queries
    // variableQueryType is the actual type (files/prefixes/sql), queryType is always 'variable'
    const variableQueryType = query.variableQueryType || 'files';
    const { prefix, filePattern, path, sqlQuery } = query;

    try {
      const response = await getBackendSrv().post(`/api/ds/query`, {
        queries: [
          {
            refId: 'variable',
            datasource: { type: this.type, uid: this.uid },
            queryType: 'variable',
            variableQueryType: variableQueryType,
            prefix: prefix || '',
            filePattern: filePattern || '*.parquet',
            path: path || '',
            sqlQuery: sqlQuery || '',
          },
        ],
      });

      const frame = response.results?.variable?.frames?.[0];
      if (!frame || !frame.data?.values?.[0]) {
        return [];
      }

      const values = frame.data.values[0] as string[];
      return values.map((value: string) => ({ text: value, value }));
    } catch {
      return [];
    }
  }

  /**
   * Parse legacy string-based variable queries
   * Supports: prefixes(path), files(path, pattern)
   */
  private async parseStringQuery(query: string): Promise<MetricFindValue[]> {
    const prefixesMatch = query.match(/^prefixes\((.*?)\)$/);
    if (prefixesMatch) {
      return this.metricFindQuery({
        queryType: 'prefixes',
        prefix: prefixesMatch[1]?.trim() || '',
      });
    }

    const filesMatch = query.match(/^files\((.*?),\s*(.*?)\)$/);
    if (filesMatch) {
      return this.metricFindQuery({
        queryType: 'files',
        prefix: filesMatch[1]?.trim() || '',
        filePattern: filesMatch[2]?.trim() || '*.parquet',
      });
    }

    // Default: treat as prefix for listing files
    return this.metricFindQuery({
      queryType: 'files',
      prefix: query.trim(),
      filePattern: '*.parquet',
    });
  }

  /**
   * List parquet files in the bucket
   */
  async listFiles(prefix = '', pattern = '*.parquet'): Promise<string[]> {
    try {
      // Use metricFindQuery which already handles the response format correctly
      const results = await this.metricFindQuery({
        variableQueryType: 'files',
        prefix: prefix,
        filePattern: pattern,
      });
      return results.map(r => r.value as string);
    } catch {
      return [];
    }
  }

  /**
   * Get schema (columns) of a parquet file
   */
  async getSchema(path: string): Promise<Array<{ name: string; type: string }>> {
    try {
      // Query the file with LIMIT 0 to just get schema
      const response = await getBackendSrv().post(`/api/ds/query`, {
        queries: [
          {
            refId: 'schema',
            datasource: { type: this.type, uid: this.uid },
            path: path,
            sqlQuery: 'SELECT * FROM parquet LIMIT 0',
          },
        ],
      });

      const frame = response.results?.schema?.frames?.[0];
      if (!frame || !frame.schema?.fields) {
        return [];
      }

      return frame.schema.fields.map((field: { name: string; type: string }) => ({
        name: field.name,
        type: field.type || 'unknown',
      }));
    } catch {
      return [];
    }
  }
}
