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
    };
  }

  filterQuery(query: MyQuery): boolean {
    // if no path has been provided, prevent the query from being executed
    return !!query.path;
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
    } catch (error) {
      console.error('Variable query failed:', error);
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
  async listFiles(prefix: string = '', pattern: string = '*.parquet'): Promise<string[]> {
    try {
      // Use metricFindQuery which already handles the response format correctly
      const results = await this.metricFindQuery({
        variableQueryType: 'files',
        prefix: prefix,
        filePattern: pattern,
      });
      return results.map(r => r.value as string);
    } catch (error) {
      console.error('Failed to list files:', error);
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
    } catch (error) {
      console.error('Failed to get schema:', error);
      return [];
    }
  }
}
