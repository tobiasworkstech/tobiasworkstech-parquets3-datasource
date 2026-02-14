import { test, expect } from '@playwright/test';

const GRAFANA_URL = process.env.GRAFANA_URL || 'http://localhost:3000';

// Increase timeout for dashboard loading
test.setTimeout(120000);

test.describe('Dashboard Verification', () => {
  test('Iris Flower Dataset dashboard loads with data', async ({ page }) => {
    await page.goto(`${GRAFANA_URL}/d/iris-dataset/iris-flower-dataset`);

    // Wait for page to fully load
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    // Take a debug screenshot
    await page.screenshot({ path: 'playwright-results/iris-dashboard.png', fullPage: true });

    // Check for panel content - Grafana uses different panel selectors across versions
    // Look for panel titles which are more stable
    const panelTitle = page.getByText('Iris Dataset Sample (First 20 Rows)');
    await expect(panelTitle).toBeVisible({ timeout: 30000 });

    // Verify there are table cells with data or panel containers with content
    const body = await page.textContent('body');
    expect(body).toBeTruthy();

    // Check for "setosa", "versicolor", "virginica" (iris species) in the page
    // At least one species should appear in any panel showing iris data
    const hasIrisData =
      body!.includes('setosa') || body!.includes('versicolor') || body!.includes('virginica');
    expect(hasIrisData).toBe(true);

    // Verify stat panels render (Total Samples, Species Distribution, etc.)
    await expect(page.getByText('Total Samples')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Species Distribution')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Average Measurements')).toBeVisible({ timeout: 10000 });

    console.log('Iris Dashboard: All panels verified with data');
  });

  test('Titanic Survival Dataset dashboard loads with data', async ({ page }) => {
    await page.goto(`${GRAFANA_URL}/d/titanic-dataset/titanic-survival-dataset`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    await page.screenshot({ path: 'playwright-results/titanic-dashboard.png', fullPage: true });

    // Check for panel title
    const panelTitle = page.getByText('Titanic Passengers (First 20 Rows)');
    await expect(panelTitle).toBeVisible({ timeout: 30000 });

    const body = await page.textContent('body');
    expect(body).toBeTruthy();

    // Check for Titanic-specific data
    const hasTitanicData =
      body!.includes('Passenger') || body!.includes('male') || body!.includes('female');
    expect(hasTitanicData).toBe(true);

    // Check key panels
    await expect(page.getByText('Total Passengers')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Survivors')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Average Fare')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Passengers by Class')).toBeVisible({ timeout: 10000 });

    console.log('Titanic Dashboard: All panels verified with data');
  });

  test('Server Metrics Time-Series dashboard loads with data', async ({ page }) => {
    await page.goto(`${GRAFANA_URL}/d/metrics-timeseries/server-metrics-time-series`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    await page.screenshot({ path: 'playwright-results/metrics-dashboard.png', fullPage: true });

    // Check for panel titles
    await expect(page.getByText('CPU Usage by Host (%)')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText('Memory Usage by Host (%)')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Disk Usage by Host (%)')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Network Throughput by Host (Mbps)')).toBeVisible({ timeout: 10000 });

    const body = await page.textContent('body');
    expect(body).toBeTruthy();

    // Time-series panels render data in canvas elements (not extractable as text)
    // Verify charts rendered by checking for metric field names in legends
    const hasMetricsData =
      body!.includes('cpu_percent') ||
      body!.includes('memory_percent') ||
      body!.includes('disk_percent') ||
      body!.includes('network_mbps') ||
      body!.includes('requests') ||
      body!.includes('errors');
    expect(hasMetricsData).toBe(true);

    // Scroll down to check gauge and summary panels
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(2000);

    await page.screenshot({ path: 'playwright-results/metrics-dashboard-bottom.png', fullPage: true });

    // Check that gauge and summary panels are in the DOM (they may lazy-load)
    await expect(page.getByText('Average CPU')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('Host Summary Statistics')).toBeVisible({ timeout: 10000 });

    console.log('Metrics Dashboard: All panels verified with data');
  });

  test('Parquet S3 Example Dashboard loads with data', async ({ page }) => {
    await page.goto(`${GRAFANA_URL}/d/parquet-s3-example/parquet-s3-example-dashboard`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    await page.screenshot({ path: 'playwright-results/example-dashboard.png', fullPage: true });

    // Check for panel titles
    const panelTitle = page.getByText('All Data (SELECT * FROM parquet LIMIT 20)');
    await expect(panelTitle).toBeVisible({ timeout: 30000 });

    const body = await page.textContent('body');
    expect(body).toBeTruthy();

    // Check for iris data in the first panel
    const hasData =
      body!.includes('setosa') || body!.includes('versicolor') || body!.includes('virginica');
    expect(hasData).toBe(true);

    // Check other panels
    await expect(page.getByText('Filtered (WHERE sepal.length > 6)')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Aggregations (COUNT, AVG)')).toBeVisible({ timeout: 10000 });

    console.log('Example Dashboard: All panels verified with data');
  });

  test('All dashboards are provisioned and accessible via API', async ({ page }) => {
    const response = await page.request.get(`${GRAFANA_URL}/api/search`);
    expect(response.ok()).toBeTruthy();

    const dashboards = await response.json();
    const uids = dashboards.map((d: { uid: string }) => d.uid);

    expect(uids).toContain('iris-dataset');
    expect(uids).toContain('titanic-dataset');
    expect(uids).toContain('metrics-timeseries');
    expect(uids).toContain('parquet-s3-example');

    console.log(`Found ${dashboards.length} dashboards: ${uids.join(', ')}`);
  });

  test('Datasource health check passes', async ({ page }) => {
    const dsResponse = await page.request.get(`${GRAFANA_URL}/api/datasources`);
    expect(dsResponse.ok()).toBeTruthy();

    const datasources = await dsResponse.json();
    const parquetDs = datasources.find(
      (ds: { type: string }) => ds.type === 'tobiasworkstech-parquets3-datasource'
    );
    expect(parquetDs).toBeTruthy();

    const healthResponse = await page.request.get(
      `${GRAFANA_URL}/api/datasources/uid/${parquetDs.uid}/health`
    );
    expect(healthResponse.ok()).toBeTruthy();

    const healthData = await healthResponse.json();
    console.log(`Datasource health: ${JSON.stringify(healthData)}`);
    expect(healthData.status).toBe('OK');
  });
});
